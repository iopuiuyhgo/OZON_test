package tests

import (
	"OZON_test/internal/handler"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			log.Printf("failed to close listener: %v", err)
		}
	}(l)
	return l.Addr().(*net.TCPAddr).Port
}

func TestHandlers(t *testing.T) {
	ip := "localhost"
	port := strconv.Itoa(findFreePort(t))
	const pathToHTML = "../page.html"

	// Чтение и шаблонизация HTML-страницы.
	tmp, err := os.ReadFile(pathToHTML)
	if err != nil {
		t.Fatalf("failed to read file %q: %v", pathToHTML, err)
	}
	data := struct {
		IP   string
		PORT string
	}{IP: ip, PORT: port}

	var pageBuffer bytes.Buffer
	err = template.Must(template.New("page").Parse(string(tmp))).Execute(&pageBuffer, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	mockStorage := newMockStorage()
	handlers := handler.CreateHandlers(MockGenerator, mockStorage, ip, port)
	err = mockStorage.Store("testOkay", "http://example.com")
	if err != nil {
		t.Errorf("cannot store %s:%s %v", "testOkay", "http://example.com", err)
	}

	go handlers.Run()
	time.Sleep(1 * time.Second)
	t.Cleanup(func() {
		handlers.Close()
	})

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	tests := []struct {
		name               string
		path               string
		method             string
		body               io.Reader
		expectedBody       string
		expectedURL        string
		expectedStatusCode int
	}{
		{
			name:               "Page",
			path:               "page",
			method:             "GET",
			body:               nil,
			expectedURL:        "",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "ValidKey",
			path:               "testOkay",
			method:             "GET",
			body:               nil,
			expectedURL:        "http://example.com",
			expectedStatusCode: http.StatusFound,
		},
		{
			name:               "ZeroKey",
			path:               "",
			method:             "GET",
			body:               nil,
			expectedURL:        "",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "InvalidKey",
			path:               "invalidT",
			method:             "GET",
			body:               nil,
			expectedURL:        "",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "InvalidKey",
			path:               "invalidT",
			method:             "GET",
			body:               nil,
			expectedURL:        "",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "ValidPost",
			path:               "",
			method:             "POST",
			body:               bytes.NewBufferString(`{"url": "http://example.com/second"}`),
			expectedURL:        "",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "DoubleValidPost",
			path:               "",
			method:             "POST",
			body:               bytes.NewBufferString(`{"url": "http://example.com/second"}`),
			expectedURL:        "",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "InvalidPost",
			path:               "",
			method:             "POST",
			body:               bytes.NewBufferString(`{}`),
			expectedURL:        "",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "BadJson",
			path:               "",
			method:             "POST",
			body:               bytes.NewBufferString(`{`),
			expectedURL:        "",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, fmt.Sprintf("http://%s:%s/%s", ip, port, tt.path), tt.body)
			assert.NoError(t, err)

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, resp.Body.Close())
			}()

			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedURL, resp.Header.Get("Location"))
		})
	}
}
