package http1

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strconv"
)

type Response struct {
	conn        net.Conn
	body        bytes.Buffer
	header      http.Header
	status      int
	wroteHeader bool
}

func NewResponse(conn net.Conn) *Response {
	return &Response{
		conn:   conn,
		header: make(http.Header),
		status: http.StatusOK,
	}
}

func (r *Response) Header() http.Header {
	return r.header
}

func (r *Response) Write(data []byte) (n int, err error) {
	return r.body.Write(data)
}

func (r *Response) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}
	r.status = statusCode
}

func (r *Response) Flush() error {
	if r.wroteHeader {
		return nil
	}

	if r.header.Get("Content-Length") == "" {
		r.header.Set("Content-Length", strconv.Itoa(r.body.Len()))
	}

	bufferWriter := bufio.NewWriter(r.conn)
	if err := r.writeHeaderBlock(bufferWriter); err != nil {
		return err
	}

	if _, err := bufferWriter.Write(r.body.Bytes()); err != nil {
		return err
	}

	return bufferWriter.Flush()
}

func (r *Response) writeHeaderBlock(w *bufio.Writer) error {
	r.wroteHeader = true

	text := http.StatusText(r.status)
	if text == "" {
		text = "Unknown"
	}

	if _, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", r.status, text); err != nil {
		return err
	}

	for key, values := range r.header {
		for _, val := range values {
			if _, err := fmt.Fprintf(w, "%s: %s\r\n", key, val); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(w, "\r\n"); err != nil {
		return err
	}

	return nil
}
