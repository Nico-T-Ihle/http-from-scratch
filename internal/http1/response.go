package http1

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type Response struct {
	conn        net.Conn
	writer      *bufio.Writer
	header      http.Header
	status      int
	wroteHeader bool
}

func NewResponse(conn net.Conn) *Response {
	return &Response{
		conn:   conn,
		writer: bufio.NewWriter(conn),
		header: make(http.Header),
		status: http.StatusOK,
	}
}

func (r *Response) Header() http.Header {
	return r.header
}

func (r *Response) Write(data []byte) (n int, err error) {
	if !r.wroteHeader {
		if err := r.writeHeaderBlock(); err != nil {
			return 0, err
		}
	}
	return r.writer.Write(data)
}

func (r *Response) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		return
	}
	r.status = statusCode
}

func (r *Response) writeHeaderBlock() error {
	r.wroteHeader = true

	text := http.StatusText(r.status)
	if text == "" {
		text = "Unknown"
	}

	for key, values := range r.header {
		for _, val := range values {
			fmt.Fprintf(r.writer, "%s: %s\r\n", key, val)
		}
	}

	return nil
}

func (r *Response) Flush() error {
	if !r.wroteHeader {
		if err := r.writeHeaderBlock(); err != nil {
			return err
		}
	}
	return r.writer.Flush()
}
