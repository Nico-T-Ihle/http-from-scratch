package http1

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

type HTTPError struct {
	StatusCode int
	Status     string
}

func (err *HTTPError) Error() string {
	return err.Status
}

func newHTTPError(code int, message string) error {
	return &HTTPError{
		StatusCode: code,
		Status:     message,
	}
}

func isValidToken(token string) bool {
	if len(token) == 0 {
		return false
	}
	for i := 0; i < len(token); i++ {
		char := token[i]
		switch {
		case char >= 'a' && char <= 'z':
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case strings.IndexByte("!#$%&'*+-.^_`|~", char) >= 0:
		default:
			return false
		}
	}
	return true
}

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if len(line) > 0 {
			return nil, newHTTPError(http.StatusBadRequest, "truncated request line")
		}
		return nil, err
	}

	if line == "\r\n" || line == "\n" {
		return nil, newHTTPError(http.StatusBadRequest, "empty request line")
	}

	if !strings.HasSuffix(line, "\r\n") {
		return nil, newHTTPError(http.StatusBadRequest, "request line must end with CRLF")
	}

	if len(line) > 8192 {
		return nil, newHTTPError(http.StatusBadRequest, "request line too long")
	}

	line = strings.TrimSuffix(line, "\r\n")

	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return nil, newHTTPError(http.StatusBadRequest, fmt.Sprintf("expected 3 tokens, got %d", len(parts)))
	}

	method, target, proto := parts[0], parts[1], parts[2]

	if !isValidToken(method) {
		return nil, newHTTPError(http.StatusBadRequest, "invalid method token")
	}

	major, minor, err := parseHTTPVersion(proto)
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.ParseRequestURI(target)
	if err != nil {
		return nil, newHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid target: %v", err))
	}

	headers := make(http.Header)
	if err := readHeaders(reader, headers); err != nil {
		return nil, err
	}

	body, contentLength, err := parseBody(headers, reader)
	if err != nil {
		return nil, err
	}

	request := &http.Request{
		Method:        method,
		URL:           parsedURL,
		Proto:         proto,
		ProtoMajor:    major,
		ProtoMinor:    minor,
		Header:        headers,
		Body:          body,
		ContentLength: contentLength,
	}

	return request, nil
}

func readHeaders(reader *bufio.Reader, headers http.Header) error {
	headerCount := 0

	for {
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			return newHTTPError(http.StatusBadRequest, "truncated header")
		}

		if !strings.HasSuffix(headerLine, "\r\n") {
			return newHTTPError(http.StatusBadRequest, "header must end with CRLF")
		}

		headerLine = strings.TrimSuffix(headerLine, "\r\n")
		if headerLine == "" {
			break
		}

		headerCount++
		if headerCount > 100 {
			return newHTTPError(http.StatusRequestHeaderFieldsTooLarge, "too many headers")
		}
		if len(headerLine) > 8192 {
			return newHTTPError(http.StatusRequestHeaderFieldsTooLarge, "header line too long")
		}

		colonIndex := strings.Index(headerLine, ":")
		if colonIndex <= 0 {
			return newHTTPError(http.StatusBadRequest, "invalid header format")
		}

		rawKey := headerLine[:colonIndex]
		if strings.ContainsAny(rawKey, " \t") {
			return newHTTPError(http.StatusBadRequest, "whitespace in header name")
		}

		if !isValidToken(rawKey) || strings.Contains(rawKey, "%") {
			return newHTTPError(http.StatusBadRequest, "invalid character in field name")
		}

		rawValue := headerLine[colonIndex+1:]

		if strings.Contains(rawValue, "\r") {
			return newHTTPError(http.StatusBadRequest, "Smuggling detected")
		}
		if strings.Contains(rawValue, "\t") {
			return newHTTPError(http.StatusBadRequest, "tab in field value")
		}

		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		headers.Add(key, value)
	}
	return nil
}

func parseBody(headers http.Header, reader *bufio.Reader) (io.ReadCloser, int64, error) {
	transferEncoding := headers.Get("Transfer-Encoding")
	contentLengthValues := headers["Content-Length"] // just wanna know if the chuck word is in!

	if transferEncoding != "" {
		if strings.EqualFold(strings.TrimSpace(transferEncoding), "chunked") {
			//Payload??
			return io.NopCloser(httputil.NewChunkedReader(reader)), -1, nil
		}
		// the others are not allowed in here!! :)
		return nil, 0, newHTTPError(http.StatusNotImplemented, "unsupported transfer encoding")
	}

	if len(contentLengthValues) > 0 {
		firstValue := strings.TrimSpace(contentLengthValues[0])

		for _, value := range contentLengthValues {
			if strings.TrimSpace(value) != firstValue {
				return nil, 0, newHTTPError(http.StatusBadRequest, "conflicting Content-Length headers")
			}
		}

		if strings.HasPrefix(firstValue, "+") || strings.HasPrefix(firstValue, "-") {
			return nil, 0, newHTTPError(http.StatusBadRequest, "invalid Content-Length sign")
		}

		length, err := strconv.ParseInt(firstValue, 10, 64)
		if err != nil || length < 0 {
			return nil, 0, newHTTPError(http.StatusBadRequest, "invalid Content-Length")
		}

		if length == 0 {
			return http.NoBody, 0, nil
		}

		return &exactReader{reader: reader, remaining: length}, length, nil
	}

	return http.NoBody, 0, nil
}

func parseHTTPVersion(proto string) (int, int, error) {
	versions := strings.Split(proto, "/")
	if len(versions) != 2 || versions[0] != "HTTP" {
		return 0, 0, newHTTPError(http.StatusBadRequest, "invalid protocol prefix")
	}

	versionParts := strings.Split(versions[1], ".")
	if len(versionParts) != 2 {
		return 0, 0, newHTTPError(http.StatusBadRequest, "malformed version")
	}

	major, errMajor := strconv.Atoi(versionParts[0])
	minor, errMinor := strconv.Atoi(versionParts[1])
	if errMajor != nil || errMinor != nil {
		return 0, 0, newHTTPError(http.StatusBadRequest, "non-numeric version")
	}

	if major == 1 && (minor == 0 || minor == 1) {
		return major, minor, nil
	}

	return 0, 0, newHTTPError(http.StatusHTTPVersionNotSupported, "version not supported")
}

type exactReader struct {
	reader    io.Reader
	remaining int64
}

// i need some think for .. unvollständigen Übertragungen
func (exactRead *exactReader) Read(p []byte) (int, error) {
	if exactRead.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > exactRead.remaining {
		p = p[:exactRead.remaining]
	}
	n, err := exactRead.reader.Read(p)
	exactRead.remaining -= int64(n)
	if err == io.EOF && exactRead.remaining > 0 {
		return n, io.ErrUnexpectedEOF
	}
	return n, err
}

func (exactRead *exactReader) Close() error {
	return nil
}
