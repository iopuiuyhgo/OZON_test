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

func (m *MockStorage) Store(key string, value string) error {
	m.data[key] = value
	return nil
}

func MockGenerator(_ string, seed int) (string, error) {
	return fmt.Sprintf("path%d", seed), nil
}

func newMockStorage() storage.Storage {
	return &MockStorage{data: make(map[string]string)}
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
	err := mockStorage.Store("key0", "http://example.com")
	if err != nil {
		t.Errorf("cannot ctore %s:%s %v", "key0", "http://example.com", err)
	}
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
