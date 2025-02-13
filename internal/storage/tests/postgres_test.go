package tests

import (
	"OZON_test/internal/storage"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"testing"
)

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
	pg, err := storage.NewPostgresStringMap(connString, tableName, 10)
	assert.NoError(t, err, "failed to create PostgresStringMap")

	key := "test_key"
	value := "http://example.com"
	err = pg.Store(key, value)
	if err != nil {
		t.Errorf("failed to store %s:%s %v", key, value, err)
	}

	loadedValue, err := pg.Load(key)
	assert.Nil(t, err, "path should exist in the database")
	assert.Equal(t, value, loadedValue, "loaded value should match stored value")

	_, err = pg.Load("nonexistent_key")
	assert.NotNil(t, err, "nonexistent path should not be found")

	err = pg.Close()
	assert.NoError(t, err, "failed to close connection")
}
