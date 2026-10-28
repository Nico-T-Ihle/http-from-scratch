package http1

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"httpServer/internal/mux1"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func ListenAndServe(addr string, handler *mux1.ServeMux) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	if handler == nil {
		handler = mux1.NewServerMux()
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error-Detected: %v", err)
			continue
		}
		println("Listening on " + conn.RemoteAddr().String())

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		go func(conn net.Conn) {
			defer cancel()
			defer conn.Close()
			HandleConnection(ctx, conn, handler)
		}(conn)
	}
}

func HandleConnection(ctx context.Context, conn net.Conn, mux *mux1.ServeMux) error {
	defer conn.Close()

	bufferReader := bufio.NewReader(conn)

	for {
		reqCtx, cancel := context.WithCancel(ctx)
		request, err := ParseRequest(bufferReader)
		if err != nil {
			cancel()
			if errors.Is(err, io.EOF) {
				return nil
			}

			statusCode := http.StatusBadRequest
			var httpErr *HTTPError
			if errors.As(err, &httpErr) {
				statusCode = httpErr.StatusCode
			}

			statusText := http.StatusText(statusCode)
			if statusText == "" {
				statusText = "Bad Request"
			}

			body := fmt.Sprintf("%d %s", statusCode, statusText)
			response := fmt.Sprintf(
				"HTTP/1.1 %d %s\r\n"+
					"Content-Type: text/plain; charset=utf-8\r\n"+
					"Connection: close\r\n"+
					"Content-Length: %d\r\n"+
					"\r\n"+
					"%s",
				statusCode, statusText, len(body), body,
			)

			conn.Write([]byte(response))
			return nil
		}

		request = request.WithContext(reqCtx)
		response := NewResponse(conn)

		// TODO: Replace the mux with mux1 rebuild!!
		mux.ServeHTTP(response, request)

		var bodyErr error
		if request.Body != nil {
			_, bodyErr = io.Copy(io.Discard, request.Body)
			request.Body.Close()

			if strings.EqualFold(request.Header.Get("Transfer-Encoding"), "chunked") && bodyErr == nil {
				for {
					trailerLine, tErr := bufferReader.ReadString('\n')
					if tErr != nil {
						bodyErr = tErr
						break
					}
					if trailerLine == "\r\n" || trailerLine == "\n" || trailerLine == "" {
						break
					}
				}
			}
		}

		if bodyErr != nil {
			cancel()
			errMsg := "400 Bad Request"
			errResp := fmt.Sprintf(
				"HTTP/1.1 400 Bad Request\r\n"+
					"Content-Type: text/plain; charset=utf-8\r\n"+
					"Connection: close\r\n"+
					"Content-Length: %d\r\n"+
					"\r\n"+
					"%s",
				len(errMsg), errMsg,
			)
			conn.Write([]byte(errResp))
			return nil
		}

		if err := response.Flush(); err != nil {
			cancel()
			return err
		}

		cancel()

		connHdr := strings.ToLower(strings.TrimSpace(request.Header.Get("Connection")))
		if (request.Proto == "HTTP/1.1" && connHdr == "close") ||
			(request.Proto == "HTTP/1.0" && connHdr != "keep-alive") {
			return nil
		}
	}
}
