package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"gorm.io/gorm"

	"go-monorepo-template/pkg/database"
	"go-monorepo-template/pkg/logger"
	productpb "go-monorepo-template/proto/acme/product/v1"
)

const (
	grpcPort = ":8080"
)

// server implements the ProductService gRPC server.
type server struct {
	productpb.UnimplementedProductServiceServer
	db  *gorm.DB
	log *slog.Logger
}

func (s *server) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.Product, error) {
	s.log.Info("GetProduct called", "product_id", req.ProductId)
	// In a real application, you would fetch the product from the database.
	return &productpb.Product{
		ProductId:   req.ProductId,
		Name:        "Test Product",
		Description: "A great product",
		Price:       99.99,
	}, nil
}

func main() {
	log := logger.New(slog.LevelDebug)
	log.Info("Starting Product API")

	db, err := database.NewClient()
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	s := &server{
		db:  db,
		log: log,
	}

	go func() {
		if err := runGrpcServer(s); err != nil {
			log.Error("gRPC server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")
}

func runGrpcServer(s *server) error {
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	productpb.RegisterProductServiceServer(grpcServer, s)

	s.log.Info("gRPC server listening", "port", grpcPort)
	return grpcServer.Serve(lis)
}
