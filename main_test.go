package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	pb "OZON_test/internal/handler/proto"
	"OZON_test/internal/storage"
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

func TestRunServer(t *testing.T) {
	port := findFreePort(t)
	ip := "localhost"
	mockStorage := newMockStorage()
	idGen := MockGenerator

	go func() {
		if err := runServer(ip, strconv.Itoa(port), mockStorage, idGen); err != nil {
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
