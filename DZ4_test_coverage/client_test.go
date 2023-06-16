package main

import (
	"net/http"          // request, response, status codes
	"net/http/httptest" // test server
	"strings"           // prefix (starts with)
	"testing"           // test subsystem
	"time"
)

// код тестирует client.go
func TestLimitOver(t *testing.T) {
	// Проверяет неотрицательность лимита сервера
	var serverCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
	}))

	req := SearchRequest{Limit: -1} // limit < 0
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Ошибка не получена")
	}

	if err.Error() != "limit must be > 0" {
		t.Errorf("Ошибка лимита сервера или иная")
	}

	if serverCalled {
		t.Errorf("Сервер не должен отвечать")
	}
}
func TestOverLimit(t *testing.T) {
	// проверяет не превышает ли лимит
	var isLimitCorrect = false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isLimitCorrect = (r.URL.Query().Get("limit") == "26")
	}))

	req := SearchRequest{Limit: 50} // лимит больше 25
	client := &SearchClient{URL: ts.URL}

	client.FindUsers(req)

	if !isLimitCorrect {
		t.Errorf("Лимит не должен превышать 25")
	}
}

func TestOffsetNonZero(t *testing.T) {
	// проверяет смещение
	var serverCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
	}))

	req := SearchRequest{Offset: -1} // задаем смещение меньше 0
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Ошибка не получена")
	}

	if err.Error() != "offset must be > 0" {
		t.Errorf("Ошибка смещения")
	}

	if serverCalled {
		t.Errorf("Сервер не должен отвечать")
	}
}

func TestTimeoutServer(t *testing.T) {
	/// время ожидания сервера
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // timeout = 5
	}))

	defer ts.Close()

	req := SearchRequest{}
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Ошибки нет")
	} else {
		// error recieve
		if !strings.HasPrefix(err.Error(), "timeout for") {
			t.Errorf("Получена ошибка ожидания")
		}
	}
}

func TestClientNonAutorized(t *testing.T) {
	req := SearchRequest{}
	client := &SearchClient{}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Ошибка не получена")
	} else {
		if !strings.HasPrefix(err.Error(), "unknown error") {
			t.Errorf("Получена неизвестная ошибка или другая")
		}
	}
}

func TestUnAuthorized(t *testing.T) {
	// ошибка отсутствия токена
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	defer ts.Close()

	req := SearchRequest{}
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Ошибка не получена")
	} else {
		if err.Error() != "Bad AccessToken" {
			t.Errorf("Получена ошибка неправильного токена")
		}
	}
}

func TestNonUnpakingJson(t *testing.T) {
	// проверка невозможности распаковки JSON файла
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("notajson"))
	}))

	defer ts.Close()

	req := SearchRequest{}
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Ошибка не получена")
	} else {
		if !strings.HasPrefix(err.Error(), "cant unpack error json") {
			t.Errorf("Невозможно распаковать JSON файл")
		}
	}
}

func TestBadOrderFeld(t *testing.T) {
	// Ошибка параметров запроса
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"ErrorBadOrderField"}`))
	}))

	defer ts.Close()

	req := SearchRequest{OrderField: "OK"}
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Ошибка не получена")
	} else {
		// error is present -> check the actual message
		if err.Error() != "OrderFeld OK invalid" {
			t.Errorf("Ошибка параметров запроса")
		}
	}
}

func TestUnknownError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"unknown"}`)) // body
	}))

	defer ts.Close()

	req := SearchRequest{OrderField: "-1"}
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Ошибка не получена")
	} else {
		if err.Error() != "unknown bad request error: unknown" {
			t.Errorf("Неизвестная ошибка системы")
		}
	}
}

func TestNoUsers(t *testing.T) {
	// Проверяет на кол-во пользователей
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`)) // пользователи отсутствуют
	}))

	defer ts.Close()

	req := SearchRequest{Limit: 5}
	client := &SearchClient{URL: ts.URL}

	res, err := client.FindUsers(req)

	if err != nil {
		t.Errorf("Ошибка не получена")
	}

	if res.NextPage {
		t.Errorf("Следующей страницы быть не может")
	}

	if len(res.Users) != 0 {
		t.Errorf("0 пользователей получено")
	}
}
func TestUsersOver(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		users := `[
      {"id":1, "name": "karla clare", "age": 25, "about": "student", "gender": "female"},
      {"id":2, "name": "andrey gonson", "age": 18, "about": "worker", "gender": "male"}
    ]`
		w.Write([]byte(users))
	}))
	defer ts.Close()

	req := SearchRequest{Limit: 1}
	client := &SearchClient{URL: ts.URL}

	res, err := client.FindUsers(req)

	if err != nil {
		t.Errorf("Ошибка не получено")
	}

	if !res.NextPage {
		t.Errorf("Должна быть еще страница")
	}

	if len(res.Users) != 1 { // and not 2
		t.Errorf("Должен быть 1 пользователь")
	}
}

func TestFatal(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer ts.Close()

	req := SearchRequest{}
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req)

	if err == nil {
		t.Errorf("Jib,rb ytn")
	} else {
		if err.Error() != "SearchServer fatal error" {
			t.Errorf("Получена ФАТАЛЬНАЯ ошибка")
		}
	}
}
