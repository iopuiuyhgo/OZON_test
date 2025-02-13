package main

import (
	"OZON_test/internal/encoder"
	"OZON_test/internal/handler"
	pb "OZON_test/internal/handler/proto"
	"OZON_test/internal/storage"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"strconv"
)

func getEnv[T any](key string, defaultValue T, parser func(string) (T, error)) T {
	if value, exists := os.LookupEnv(key); exists {
		parsedValue, err := parser(value)
		if err == nil {
			return parsedValue
		}
	}
	return defaultValue
}

func idString(key string) (string, error) {
	return key, nil
}

func main() {
	ip := getEnv("SERVER_IP", "localhost", idString)
	port := getEnv("SERVER_PORT", "8080", idString)
	inMemory := getEnv("USE_IN_MEMORY", true, strconv.ParseBool)
	postgresPath := getEnv("POSTGRES_PATH", "", idString)
	tableName := getEnv("TABLE_NAME", "", idString)
	grpcInterface := getEnv("GRPC", true, strconv.ParseBool)
	keyLen := getEnv("KEY_LEN", 10, strconv.Atoi)

	idGen := func(url string, seed int) (string, error) { return encoder.GenerateSecureShortId(url, seed, keyLen) }

	var (
		storageMap storage.Storage
		err        error
	)

	if inMemory {
		storageMap = storage.NewSafeMap()
	} else {
		storageMap, err = storage.NewPostgresStringMap(postgresPath, tableName, keyLen)
	}
	if err != nil {
		log.Println("Error: No valid storage configuration provided. Please specify either in-memory storage or a valid PostgreSQL path.")
		log.Fatalln(err)
		return
	}

	if err != nil {
		log.Fatalf(err.Error())
		return
	}
	if grpcInterface {
		if err := runServer(ip, port, storageMap, idGen); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	} else {
		h := handler.CreateHandlers(idGen, storageMap, ip, port)
		h.Run()
	}
}

func runServer(ip string, port string, storage storage.Storage, idGen func(url string, seed int) (string, error)) error {
	server := grpc.NewServer()
	pb.RegisterUrlServiceServer(server, handler.NewUrlServer(idGen, &storage, ip))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	log.Printf("gRPC server started on port %s", port)
	return server.Serve(lis)
}
