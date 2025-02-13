package tests

import (
	"OZON_test/internal/handler"
	pb "OZON_test/internal/handler/proto"
	"OZON_test/internal/storage"
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MockStorage struct {
	data map[string]string
}

func (m *MockStorage) Load(key string) (string, error) {
	value, ok := m.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return value, nil
}

func (m *MockStorage) Store(key, value string) error {
	m.data[key] = value
	return nil
}

func MockGenerator(_ string, seed int) (string, error) {
	return fmt.Sprintf("path%d", seed), nil
}

func newMockStorage() storage.Storage {
	return &MockStorage{data: make(map[string]string)}
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
		})
	}
}

func TestUrlServer_Redirect(t *testing.T) {
	mockStorage := newMockStorage()
	err := mockStorage.Store("validKey", "http://example.com")
	if err != nil {
		t.Errorf("cannot store %s:%s %v", "validKey", "http://example.com", err)
	}

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
