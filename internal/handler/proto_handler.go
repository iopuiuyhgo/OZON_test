package handler

import (
	"context"
	"fmt"

	pb "OZON_test/internal/handler/proto"
	"OZON_test/internal/storage"
)

type UrlServer struct {
	pb.UnimplementedUrlServiceServer
	generator func(url string, seed int) (string, error)
	storage   *storage.Storage
	ip        string
}

func NewUrlServer(generator func(url string, seed int) (string, error), storage *storage.Storage, ip string) *UrlServer {
	return &UrlServer{generator: generator, storage: storage, ip: ip}
}

func (s *UrlServer) GenerateKey(_ context.Context, req *pb.GenerateKeyRequest) (*pb.GenerateKeyResponse, error) {
	url := req.GetUrl()
	if url == "" {
		return nil, fmt.Errorf("missing URL parameter")
	}

	var res string
	for i := 0; ; i++ {
		key, err := s.generator(url, i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key: %v", err)
		}

		v, ok := (*s.storage).Load(key)
		if !ok {
			res = key
			(*s.storage).Store(key, url)
			break
		}
		if v == url {
			return &pb.GenerateKeyResponse{
				Message:  "Data already received",
				ShortUrl: key,
			}, nil
		}
	}

	return &pb.GenerateKeyResponse{
		Message:  "Data received successfully",
		ShortUrl: res,
	}, nil
}

func (s *UrlServer) Redirect(_ context.Context, req *pb.RedirectRequest) (*pb.RedirectResponse, error) {
	key := req.GetKey()
	if key == "" {
		return nil, fmt.Errorf("missing key parameter")
	}

	redirectURL, ok := (*s.storage).Load(key)
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	return &pb.RedirectResponse{
		Url: redirectURL,
	}, nil
}
