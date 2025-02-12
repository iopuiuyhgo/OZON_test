package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"

	"OZON_test/internal/handler"
	pb "OZON_test/internal/handler/proto"
	"OZON_test/internal/storage"
)

// MockStorage реализует интерфейс Storage для тестов.
type MockStorage struct {
	data map[string]string
}

func (m *MockStorage) Load(key string) (string, bool) {
	value, ok := m.data[key]
	return value, ok
}

func (m *MockStorage) Store(key, value string) {
	m.data[key] = value
}

func MockGenerator(url string, seed int) (string, error) {
	return fmt.Sprintf("path%d", seed), nil
}

func newMockStorage() storage.Storage {
	return &MockStorage{data: make(map[string]string)}
}

func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestGenerateKey(t *testing.T) {
	mockStorage := newMockStorage()

	server := handler.NewUrlServer(MockGenerator, &mockStorage, "localhost")

	tests := []struct {
		name          string
		url           string
		expectedError bool
		expectedMsg   string
	}{
		{
			name:          "Successful path generation",
			url:           "http://example.com",
			expectedError: false,
			expectedMsg:   "Data received successfully",
		},
		{
			name:          "URL already exists",
			url:           "http://example.com",
			expectedError: false,
			expectedMsg:   "Data already received",
		},
		{
			name:          "Empty URL",
			url:           "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // захват переменной
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.GenerateKeyRequest{Url: tt.url}
			resp, err := server.GenerateKey(context.Background(), req)
			if tt.expectedError {
				assert.Error(t, err, "expected error for url: %q", tt.url)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMsg, resp.Message)
		})
	}
}

func TestRedirect(t *testing.T) {
	mockStorage := newMockStorage()
	mockStorage.Store("key0", "http://example.com")
	server := handler.NewUrlServer(nil, &mockStorage, "localhost")

	tests := []struct {
		name          string
		key           string
		expectedURL   string
		expectedError bool
	}{
		{
			name:          "Successful redirect",
			key:           "key0",
			expectedURL:   "http://example.com",
			expectedError: false,
		},
		{
			name:          "Empty path",
			key:           "",
			expectedError: true,
		},
		{
			name:          "Key not found",
			key:           "nonexistent",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // захват переменной
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.RedirectRequest{Key: tt.key}
			resp, err := server.Redirect(context.Background(), req)
			if tt.expectedError {
				assert.Error(t, err, "expected error for key: %q", tt.key)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedURL, resp.Url)
		})
	}
}

func TestRunServer(t *testing.T) {
	port := findFreePort(t)
	ip := "localhost"
	mockStorage := newMockStorage()

	go func() {
		if err := runServer(ip, strconv.Itoa(port), mockStorage); err != nil {
			t.Errorf("failed to start server: %v", err)
		}
	}()

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", ip, port), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	t.Cleanup(func() {
		assert.NoError(t, conn.Close())
	})

	client := pb.NewUrlServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.GenerateKey(ctx, &pb.GenerateKeyRequest{Url: "http://example.com"})
	assert.NoError(t, err, "failed to call GenerateKey")
}

func setupPostgresContainer(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpassword",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	connString := fmt.Sprintf("postgres://testuser:testpassword@%s:%s/testdb", host, port.Port())
	teardown := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}
	return connString, teardown
}

func TestPostgresStringMap(t *testing.T) {
	connString, teardown := setupPostgresContainer(t)
	defer teardown()

	tableName := "test_table"
	pg, err := storage.NewPostgresStringMap(connString, tableName)
	assert.NoError(t, err, "failed to create PostgresStringMap")

	key := "test_key"
	value := "http://example.com"
	pg.Store(key, value)

	loadedValue, ok := pg.Load(key)
	assert.True(t, ok, "path should exist in the database")
	assert.Equal(t, value, loadedValue, "loaded value should match stored value")

	_, ok = pg.Load("nonexistent_key")
	assert.False(t, ok, "nonexistent path should not be found")

	err = pg.Close()
	assert.NoError(t, err, "failed to close connection")
}

func TestHandlers(t *testing.T) {
	ip := "localhost"
	port := strconv.Itoa(findFreePort(t))
	const pathToHTML = "./internal/handler/page.html"

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
	handlers := handler.CreateHandlers(MockGenerator, &mockStorage, ip, port)
	mockStorage.Store("testOkay", "http://example.com")

	go handlers.Run()
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
			expectedBody:       pageBuffer.String(),
			expectedURL:        "",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "ValidKey",
			path:               "testOkay",
			method:             "GET",
			body:               nil,
			expectedBody:       "<a href=\"http://example.com\">Found</a>.\n\n",
			expectedURL:        "http://example.com",
			expectedStatusCode: http.StatusFound,
		},
		{
			name:               "ZeroKey",
			path:               "",
			method:             "GET",
			body:               nil,
			expectedBody:       "Missing key parameter\n",
			expectedURL:        "",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "InvalidKey",
			path:               "invalidT",
			method:             "GET",
			body:               nil,
			expectedBody:       "Key not found\n",
			expectedURL:        "",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "InvalidKey",
			path:               "invalidT",
			method:             "GET",
			body:               nil,
			expectedBody:       "Key not found\n",
			expectedURL:        "",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "ValidPost",
			path:               "",
			method:             "POST",
			body:               bytes.NewBufferString(`{"url": "http://example.com/second"}`),
			expectedBody:       fmt.Sprintf("{\"URL\":\"http://%s:%s/path0\",\"message\":\"Data received successfully\"}\n", ip, port),
			expectedURL:        "",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "DoubleValidPost",
			path:               "",
			method:             "POST",
			body:               bytes.NewBufferString(`{"url": "http://example.com/second"}`),
			expectedBody:       fmt.Sprintf("{\"URL\":\"http://%s:%s/path0\",\"message\":\"Data already received\"}\n", ip, port),
			expectedURL:        "",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "InvalidPost",
			path:               "",
			method:             "POST",
			body:               bytes.NewBufferString(`{}`),
			expectedBody:       "Missing url parameter\n",
			expectedURL:        "",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "BadJson",
			path:               "",
			method:             "POST",
			body:               bytes.NewBufferString(`{`),
			expectedBody:       "Invalid JSON format\n",
			expectedURL:        "",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		tt := tt // захват переменной
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, fmt.Sprintf("http://%s:%s/%s", ip, port, tt.path), tt.body)
			assert.NoError(t, err)

			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer func() {
				assert.NoError(t, resp.Body.Close())
			}()

			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, string(bodyBytes))

			// Проверка заголовка Location для редиректа.
			assert.Equal(t, tt.expectedURL, resp.Header.Get("Location"))
		})
	}
}

func TestUrlServer_GenerateKey(t *testing.T) {
	mockStorage := newMockStorage()
	server := handler.NewUrlServer(MockGenerator, &mockStorage, "localhost")

	tests := []struct {
		name        string
		url         string
		wantMessage string
		wantKey     string
		wantErr     bool
	}{
		{
			name:        "New URL",
			url:         "http://example.com",
			wantMessage: "Data received successfully",
			wantKey:     "path0",
			wantErr:     false,
		},
		{
			name:        "Duplicate URL",
			url:         "http://example.com",
			wantMessage: "Data already received",
			wantKey:     "path0",
			wantErr:     false,
		},
		{
			name:        "Empty URL",
			url:         "",
			wantMessage: "",
			wantKey:     "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		tt := tt // захват переменной
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.GenerateKeyRequest{Url: tt.url}
			resp, err := server.GenerateKey(context.Background(), req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMessage, resp.Message)
			assert.Equal(t, tt.wantKey, resp.ShortUrl)

			// Для "New URL" проверяем, что хранилище обновилось.
			if tt.name == "New URL" {
				val, ok := mockStorage.Load(tt.wantKey)
				assert.True(t, ok)
				assert.Equal(t, tt.url, val)
			}
		})
	}
}

func TestUrlServer_Redirect(t *testing.T) {
	mockStorage := newMockStorage()
	mockStorage.Store("validKey", "http://example.com")

	server := handler.NewUrlServer(nil, &mockStorage, "localhost")

	tests := []struct {
		name    string
		key     string
		wantUrl string
		wantErr bool
	}{
		{
			name:    "Valid Key",
			key:     "validKey",
			wantUrl: "http://example.com",
			wantErr: false,
		},
		{
			name:    "Invalid Key",
			key:     "invalidKey",
			wantUrl: "",
			wantErr: true,
		},
		{
			name:    "Empty Key",
			key:     "",
			wantUrl: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt // захват переменной
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.RedirectRequest{Key: tt.key}
			resp, err := server.Redirect(context.Background(), req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantUrl, resp.Url)
		})
	}
}

func TestGenerateKey_GeneratorError(t *testing.T) {
	mockStorage := newMockStorage()
	errorGenerator := func(url string, seed int) (string, error) {
		return "", fmt.Errorf("generation error")
	}
	server := handler.NewUrlServer(errorGenerator, &mockStorage, "localhost")

	req := &pb.GenerateKeyRequest{Url: "http://example.com"}
	_, err := server.GenerateKey(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate key")
}
