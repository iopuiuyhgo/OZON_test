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

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func main() {
	ip := getEnv("SERVER_IP", "localhost")
	port := getEnv("SERVER_PORT", "8080")
	inMemory := getEnvBool("USE_IN_MEMORY", true)
	postgresPath := getEnv("POSTGRES_PATH", "")
	tableName := getEnv("TABLE_NAME", "")
	grpcInterface := getEnvBool("GRPC", true)

	idGen := encoder.GenerateSecureShortId

	var (
		storageMap storage.Storage
		err        error
	)

	if inMemory {
		storageMap = storage.NewSafeMap()
	} else {
		storageMap, err = storage.NewPostgresStringMap(postgresPath, tableName)
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
		if err := runServer(ip, port, storageMap); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	} else {
		h := handler.CreateHandlers(idGen, &storageMap, ip, port)
		h.Run()
	}
}

func runServer(ip string, port string, storage storage.Storage) error {
	idGen := encoder.GenerateSecureShortId
	server := grpc.NewServer()
	pb.RegisterUrlServiceServer(server, handler.NewUrlServer(idGen, &storage, ip))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	log.Printf("gRPC server started on port %s", port)
	return server.Serve(lis)
}
