package request

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

var errRequestMalformed = errors.New("error request malformed")

type Request struct {
	Method     string
	Target     string
	Path       string
	Query      string
	Version    string
	Headers    Header
	Body       []byte
	RemoteAddr string
}

// String implements fmt.Stringer to provide a beautifully formatted
// string representation of the HTTP Request.
func (r *Request) String() string {
	var sb strings.Builder

	sb.WriteString("\n=== HTTP Request ===\n")
	sb.WriteString(fmt.Sprintf("%s %s %s\n", r.Method, r.Target, r.Version))
	sb.WriteString(fmt.Sprintf("RemoteAddr: %s\n", r.RemoteAddr))

	if r.Path != "" {
		sb.WriteString(fmt.Sprintf("Path:       %s\n", r.Path))
	}
	if r.Query != "" {
		sb.WriteString(fmt.Sprintf("Query:      %s\n", r.Query))
	}

	sb.WriteString("\nHeaders:\n")
	if len(r.Headers) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for k, values := range r.Headers {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, strings.Join(values, ", ")))
		}
	}

	if len(r.Body) > 0 {
		sb.WriteString(fmt.Sprintf("\nBody (%d bytes):\n", len(r.Body)))
		// For a simple lab server, converting body to string is fine to see the content!
		sb.WriteString(string(r.Body))
		sb.WriteString("\n")
	}
	sb.WriteString("====================")

	return sb.String()
}

type Header map[string][]string

var httpHeaderEndLine = "\r\n"

func Read(conn net.Conn) (*Request, error) {
	reader := bufio.NewReader(conn)
	req := &Request{
		Headers:    make(Header),
		RemoteAddr: conn.RemoteAddr().String(),
	}

	if err := readRequestLine(req, reader); err != nil {
		return nil, fmt.Errorf("first line read: %w", err)
	}

	if err := readHeader(req, reader); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	if err := readBody(req, reader); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return req, nil
}

func readRequestLine(req *Request, reader *bufio.Reader) error {
	str, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("first line: %w", err)
	}
	requestLineSplit := strings.Split(str, " ")
	if len(requestLineSplit) != 3 {
		return errRequestMalformed
	}

	if _, ok := methodMap[requestLineSplit[0]]; !ok {
		return errRequestMalformed
	}

	req.Method = requestLineSplit[0]
	req.Target = requestLineSplit[1]

	targetSplit := strings.Split(req.Target, "?")

	if len(targetSplit) == 0 {
		return errRequestMalformed
	}

	req.Path = targetSplit[0]

	if len(targetSplit) > 1 {
		req.Query = targetSplit[1]
	}

	return nil
}

func readHeader(req *Request, reader *bufio.Reader) error {
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("read header: %w", err)
		}

		if str == httpHeaderEndLine {
			break
		}

		headerSplit := strings.SplitN(str, ":", 2)

		if len(headerSplit) != 2 {
			return errRequestMalformed
		}

		req.Headers[strings.ToLower(headerSplit[0])] = append(req.Headers[strings.ToLower(headerSplit[0])], strings.TrimSpace(headerSplit[1]))
	}

	return nil
}

func readBody(req *Request, reader *bufio.Reader) error {
	if len(req.Headers["content-length"]) == 0 {
		return nil
	}

	contentLength, err := strconv.Atoi(req.Headers["content-length"][len(req.Headers["content-length"])-1])
	if err != nil {
		return fmt.Errorf("parse int content length: %w", err)
	}

	bodyByte := make([]byte, contentLength)

	if _, err := io.ReadFull(reader, bodyByte); err != nil {
		return fmt.Errorf("error read body byte: %w", err)
	}

	req.Body = bodyByte

	return nil
}
