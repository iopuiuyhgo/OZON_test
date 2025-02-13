package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log"
	"strconv"
)

type PostgresStringMap struct {
	conn      *pgx.Conn
	tableName string
}

func NewPostgresStringMap(connString string, tableName string, keyLen int) (*PostgresStringMap, error) {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}

	q, err := checkTableExists(conn, tableName)
	if err != nil {
		return nil, err
	}

	if !q {
		err := createTable(conn, tableName, keyLen)
		if err != nil {
			return nil, err
		}
	}

	return &PostgresStringMap{conn: conn, tableName: tableName}, nil
}

func (pg *PostgresStringMap) Load(key string) (value string, err error) {
	query := fmt.Sprintf(`
        SELECT url
        FROM "%s"
        WHERE id = $1
    `, pg.tableName)

	var url string
	err = pg.conn.QueryRow(context.Background(), query, key).Scan(&url)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		log.Printf("Error loading key: %v", err)
		return "", err
	}
	return url, nil
}

func (pg *PostgresStringMap) Store(key string, value string) error {
	query := fmt.Sprintf(`
        INSERT INTO "%s" (id, url)
        VALUES ($1, $2)
        ON CONFLICT (id) DO UPDATE SET url = EXCLUDED.url
    `, pg.tableName)

	_, err := pg.conn.Exec(context.Background(), query, key, value)
	if err != nil {
		log.Printf("Error storing key: %v", err)
		return err
	}
	return nil
}

func (pg *PostgresStringMap) Close() error {
	return pg.conn.Close(context.Background())
}

func checkTableExists(conn *pgx.Conn, tableName string) (bool, error) {
	var exists bool
	query := `
        SELECT EXISTS (
            SELECT 1
            FROM information_schema.tables
            WHERE table_name = $1
        );
    `
	err := conn.QueryRow(context.Background(), query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	return exists, nil
}

func createTable(conn *pgx.Conn, tableName string, keyLen int) error {
	query := fmt.Sprintf(`
        CREATE TABLE "%s" (
            id CHAR(%s) PRIMARY KEY,
    		url TEXT NOT NULL
        );
    `, tableName, strconv.Itoa(keyLen))

	_, err := conn.Exec(context.Background(), query)
	if err != nil {
		return err
	}
	return nil
}
