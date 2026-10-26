package http1

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	line = strings.TrimSpace(line)

	checkLength := strings.Split(line, " ")
	if len(checkLength) != 3 {
		return nil, fmt.Errorf("expected 3 tokens, got %d", len(checkLength))
	}

	var methods [1]string
	var path [1]string
	var protocol [1]string

	for i := range checkLength {
		switch i {
		case 0:
			methods[0] = checkLength[i]
		case 1:
			path[0] = checkLength[i]
		case 2:
			protocol[0] = checkLength[i]
		}
	}

	parsedURL, err := url.ParseRequestURI(path[0])
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	h, err2 := headerReader(reader, headers)
	if err2 != nil {
		return h, err2
	}

	body, contentLength, err := parseBody(headers, reader)
	if err != nil {
		return nil, err
	}

	request := &http.Request{
		Method:        methods[0],
		URL:           parsedURL,
		Proto:         protocol[0],
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        headers,
		Body:          body,
		ContentLength: contentLength,
	}

	return request, nil
}

func headerReader(reader *bufio.Reader, headers http.Header) (*http.Request, error) {
	println("headers", headers)
	for {
		// need to go through the body!!
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading header: %w", err)
		}

		headerLine = strings.TrimRight(headerLine, "\r\n")
		println("headerLine", headerLine)
		if headerLine == "" {
			break
		}

		colonIndex := strings.Index(headerLine, ":")
		if colonIndex == -1 {
			return nil, fmt.Errorf("malformed header line: %q", headerLine)
		}

		key := strings.TrimSpace(headerLine[:colonIndex])
		value := strings.TrimSpace(headerLine[colonIndex+1:])

		headers.Add(key, value)
	}
	return nil, nil
}

func parseBody(headers http.Header, reader *bufio.Reader) (io.ReadCloser, int64, error) {

	transferEncoding := headers.Get("Transfer-Encoding")
	contentLengthStr := headers.Get("Content-Length")

	if contentLengthStr != "" && transferEncoding != "" {
		return nil, 0, fmt.Errorf("400 Bad Request")
	}

	if strings.EqualFold(transferEncoding, "chunked") {
		return nil, 0, fmt.Errorf("501 Not Implemented")
	} else if len(contentLengthStr) > 0 {
		contentLengthIntValue, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			fmt.Errorf(err.Error())
		}

		contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil || contentLength < 0 {
			return nil, 0, fmt.Errorf("400 Bad Request: invalid Content-Length: %q", contentLengthStr)
		}

		if contentLengthIntValue > 0 {
			limitedReader := io.LimitReader(reader, contentLength)
			return io.NopCloser(limitedReader), contentLength, nil
		}

		return http.NoBody, 0, nil
	}

	return http.NoBody, 0, nil
}

// Todo: checking if the 400 works general testing!!
// github login ssh
