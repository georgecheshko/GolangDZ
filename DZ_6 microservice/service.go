package main

// Импортируем пакеты
import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"log"
	"net"
	"strings"
	"time"
)

// BizSrv базовая структура
type BizSrv struct {
}

// Зададим функции обработчики запросов
func (s BizSrv) Add(ctx context.Context, _ *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s BizSrv) Check(ctx context.Context, _ *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s BizSrv) Test(ctx context.Context, _ *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

// Структурка для запроса
type Visit struct {
	Method   string
	Consumer string
}

// Структура для сбора статистики
type AdmSrv struct {
	ctx context.Context

	// логгирование
	logChan        chan *Event // канал логгирование события
	logSubChan     chan chan *Event
	logSubscribers []chan *Event // список подписчиков

	statChan        chan *Visit
	statSubChan     chan chan *Visit
	statSubscribers []chan *Visit
}

// Структура на методы доступа для пользователей
type ACL map[string][]string

// общая структурка
type Srv struct {
	acl ACL

	BizSrv
	AdmSrv
}

// Создаем канал для логов , добавялем в список участников.
func (s *AdmSrv) Logging(_ *Nothing, srv Admin_LoggingServer) error {

	ch := make(chan *Event, 0)
	s.logSubChan <- ch

	for {
		select {
		case event := <-ch:
			srv.Send(event)
		case <-s.ctx.Done():
			return nil
		}
	}
}

// Создаем канал для статистики.После нового события, увеличиваем счетчики.
func (s AdmSrv) Statistics(interval *StatInterval, srv Admin_StatisticsServer) error {
	ch := make(chan *Visit, 0)
	s.statSubChan <- ch

	period := time.Second * time.Duration(interval.IntervalSeconds)
	ticker := time.NewTicker(period)
	stat := &Stat{
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}
	for {
		select {
		case v := <-ch:
			stat.ByMethod[v.Method] += 1
			stat.ByConsumer[v.Consumer] += 1
		case <-ticker.C:
			// отправялем статиктику
			srv.Send(stat)
			// откатываем статистику
			stat.ByMethod = map[string]uint64{}
			stat.ByConsumer = map[string]uint64{}
		}
	}
}

// Сервис

func StartMyMicroservice(ctx context.Context, listenAddr string, aclData string) error {
	// парсим порт и ACL
	var acl ACL
	if err := json.Unmarshal([]byte(aclData), &acl); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", listenAddr) // слушаем адрес-порт
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}

	// Формируем имя хоста и порт
	srv := &Srv{AdmSrv: AdmSrv{ctx: ctx}, acl: acl}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(srv.unaryInterceptor),
		grpc.StreamInterceptor(srv.streamInterceptor),
	)

	// Регистрируем 2 сервера
	RegisterBizServer(grpcServer, srv)
	RegisterAdminServer(grpcServer, srv)

	// Создаем каналы статистики
	srv.logChan = make(chan *Event, 0)
	srv.logSubChan = make(chan chan *Event, 0)

	srv.statChan = make(chan *Visit, 0)
	srv.statSubChan = make(chan chan *Visit, 0)
	// Сформируеем функции для логгирования

	go func() {
		for {
			select {
			case event := <-srv.logChan: // новое событие

				// уведомление о новом событии
				for _, subChan := range srv.logSubscribers {
					subChan <- event
				}
			case newSub := <-srv.logSubChan:
				// Добавялем в лист подписчиков
				srv.logSubscribers = append(srv.logSubscribers, newSub)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Аналогично для сбора статистики
	go func() {
		for {
			select {
			case stat := <-srv.statChan:

				for _, statChan := range srv.statSubscribers {
					statChan <- stat
				}
			case newSub := <-srv.statSubChan:
				srv.statSubscribers = append(srv.statSubscribers, newSub)
			case <-ctx.Done():
				return
			}
		}
	}()

	// стартуем сервев grpc
	go func() {
		err = grpcServer.Serve(lis)

	}()

	// функция остановки сервера по сигналу ctx
	go func() {
		<-ctx.Done()
		grpcServer.Stop()
	}()

	return nil
}

func (s *Srv) checkPermissions(ctx context.Context, method string) error {
	// получаем данный о подключении ctx
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "Не удалось получить метаданные")
	}

	consumer, ok := meta["consumer"]
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "не удалось получить метаданные")
	}

	// проверка наличия прав
	methods, ok := s.acl[consumer[0]]
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "Нет в списке авторизованных пользователей")
	}

	// получаем имя метода из исходного инпута
	methodName := func(input string) string {
		methodParts := strings.Split(input, "/")
		return methodParts[len(methodParts)-1]
	}

	// Получаем запрашиваемый метод
	reqMethodName := methodName(method)
	for _, method := range methods {
		methodName := methodName(method)

		// Если доступ есть то все норм
		if methodName == "*" || methodName == reqMethodName {
			return nil
		}
	}
	// Иначе нет
	return grpc.Errorf(codes.Unauthenticated, "Нет в списке авторизованных пользователей")
}

func (s *Srv) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	//Одиночнео подключение
	// Проверяем доступ из предыдущей функции
	if err := s.checkPermissions(ctx, info.FullMethod); err != nil {
		return nil, err
	}
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "Не авторизован")
	}

	consumer, ok := meta["consumer"]
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "Не авторизован")
	}
	// логгируем событие
	s.logChan <- &Event{
		Consumer: consumer[0],
		Method:   info.FullMethod,
		Host:     "127.0.0.1:8083",
	}
	s.statChan <- &Visit{Method: info.FullMethod, Consumer: consumer[0]}

	return handler(ctx, req)
}

func (s *Srv) streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// Потоковое подключение
	if err := s.checkPermissions(ss.Context(), info.FullMethod); err != nil {
		return err
	}

	meta, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "can't get metadata")
	}

	consumer, ok := meta["consumer"]
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "can't get metadata")
	}

	// Логгирование события
	s.logChan <- &Event{
		Consumer: consumer[0],
		Method:   info.FullMethod,
		Host:     "127.0.0.1:8083",
	}
	s.statChan <- &Visit{Method: info.FullMethod, Consumer: consumer[0]}

	return handler(srv, ss)
}
