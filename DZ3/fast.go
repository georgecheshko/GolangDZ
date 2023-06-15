package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	// easyjson
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// known structure(s) to avoid interfaces
type User struct {
	Email string
	Name  string
	// list of browsers
	Browsers []string
}

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	// SlowSearch(out)
	// return

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	// ---

	// ~set
	// O(1) access time
	browsers := make(map[string]bool, 120)
	// file reader declaration
	reader := bufio.NewReader(file)

	fmt.Fprintln(out, "found users:")

	i := -1 // index
	user := &User{}
	for { // go line-by-line
		i++
		line, _, err := reader.ReadLine() //.ReadBytes('\n')
		if err != nil {
			if err == io.EOF { // end of file?
				break // stop right here
			} else {
				panic(err)
			}
		}

		// process the line
		err = user.UnmarshalJSON(line)
		if err != nil {
			panic(err)
		}

		// process browsers
		androidDetected := false
		msieDetected := false
		for _, browser := range user.Browsers {
			// fmt.Println(browser)
			if strings.Contains(browser, "Android") {
				androidDetected = true
				browsers[browser] = true
			} else if strings.Contains(browser, "MSIE") {
				msieDetected = true
				browsers[browser] = true
			}
		}

		// both 'Android' and 'MSIE' must be present
		if !(androidDetected && msieDetected) {
			continue // neither Android nor MSIE -> skip
		}

		// process the email address
		email := strings.Replace(user.Email, "@", " [at] ", 1) // hopefully, only one '@' sign
		fmt.Fprintf(out, "[%d] %s <%s>\n", i, user.Name, email)
	}
	fmt.Fprintln(out) // blank newline
	fmt.Fprintln(out, "Total unique browsers", len(browsers))
}

// to be executable
// func main() {
// 	FastSearch(os.Stdout)
// }

// ===

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjson3486653aDecodeCourseraGoWsHw3Bench(in *jlexer.Lexer, out *User) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "email":
			out.Email = string(in.String())
		case "name":
			out.Name = string(in.String())
		case "browsers":
			if in.IsNull() {
				in.Skip()
				out.Browsers = nil
			} else {
				in.Delim('[')
				if out.Browsers == nil {
					if !in.IsDelim(']') {
						out.Browsers = make([]string, 0, 4)
					} else {
						out.Browsers = []string{}
					}
				} else {
					out.Browsers = (out.Browsers)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = string(in.String())
					out.Browsers = append(out.Browsers, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson3486653aEncodeCourseraGoWsHw3Bench(out *jwriter.Writer, in User) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"Email\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Email))
	}
	{
		const prefix string = ",\"Name\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Name))
	}
	{
		const prefix string = ",\"Browsers\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Browsers == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v2, v3 := range in.Browsers {
				if v2 > 0 {
					out.RawByte(',')
				}
				out.String(string(v3))
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}

func (v User) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson3486653aEncodeCourseraGoWsHw3Bench(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

func (v User) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson3486653aEncodeCourseraGoWsHw3Bench(w, v)
}

func (v *User) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson3486653aDecodeCourseraGoWsHw3Bench(&r, v)
	return r.Error()
}

func (v *User) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson3486653aDecodeCourseraGoWsHw3Bench(l, v)
}
