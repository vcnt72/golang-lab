package response

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type ResponseWriter interface {
	WriteHeader(statusCode int)
	Headers() Headers
	Write(body []byte) (int, error)
	Flush() (int, error)
}

var (
	HeaderContentLength = "Content-Length"
	HeaderContentType   = "Content-Type"
)

type Headers map[string][]string

func (h Headers) Set(key string, value string) {
	h[key] = []string{value}
}

type responseWriter struct {
	headers    Headers
	statusCode int
	conn       io.Writer
	bodyBuff   bytes.Buffer
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseWriter) Headers() Headers {
	return r.headers
}

func (r *responseWriter) Write(body []byte) (int, error) {
	return r.bodyBuff.Write(body)
}

func (r *responseWriter) Flush() (int, error) {
	r.headers[HeaderContentLength] = []string{strconv.Itoa(r.bodyBuff.Len())}
	var sb strings.Builder

	fmt.Fprintf(&sb, "HTTP/1.1 %d %s\r\n", r.statusCode, http.StatusText(r.statusCode))

	for k, values := range r.headers {
		for _, v := range values {
			fmt.Fprintf(&sb, "%s: %s\r\n", k, v)
		}
	}

	fmt.Fprintf(&sb, "\r\n")
	sb.Write(r.bodyBuff.Bytes())

	return r.conn.Write([]byte(sb.String()))
}

func NewWriter(conn io.Writer) ResponseWriter {
	return &responseWriter{
		conn:       conn,
		headers:    make(Headers),
		statusCode: http.StatusOK,
	}
}
