package http1

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func ListenAndServe(addr string, handler http.Handler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	if handler == nil {
		handler = http.DefaultServeMux
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error-Detected: %v", err)
			continue
		}
		println("Listening on " + conn.RemoteAddr().String())

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		go func(c net.Conn) {
			defer cancel()
			defer c.Close()
			HandleConnection(ctx, c, handler)
		}(conn)
	}
}

func HandleConnection(ctx context.Context, conn net.Conn, mux http.Handler) error {
	// i think i need to add a loop or something like this?!
	defer conn.Close()

	reqCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	bufferReader := bufio.NewReader(conn)
	request, err := ParseRequest(bufferReader)
	if err != nil {
		response := "HTTP/1.1 400 Bad Request\r\n" +
			"Content-Type: text/plain; charset=utf-8\r\n" +
			"Connection: close\r\n" +
			"Content-Length: 15\r\n" +
			"\r\n" +
			"400 Bad Request"

		conn.Write([]byte(response))

		return err
	}
	request = request.WithContext(reqCtx)
	response := NewResponse(conn)

	mux.ServeHTTP(response, request)

	if err := response.Flush(); err != nil {
		log.Printf("flush error: %v", err)
	}

	return nil
}
