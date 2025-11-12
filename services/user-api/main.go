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
	userpb "go-monorepo-template/proto/acme/user/v1"
)

const (
	grpcPort = ":8080"
)

// server implements the UserService gRPC server.
type server struct {
	userpb.UnimplementedUserServiceServer
	db  *gorm.DB
	log *slog.Logger
}

func (s *server) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.User, error) {
	if s.log != nil {
		s.log.Info("GetUser called", "user_id", req.UserId)
	}
	// In a real application, you would fetch the user from the database.
	// We'll return a mock user for this example.
	return &userpb.User{
		UserId:      req.UserId,
		DisplayName: "Test User",
		Email:       "test@example.com",
	}, nil
}

func main() {
	log := logger.New(slog.LevelDebug)
	log.Info("Starting User API")

	// Initialize database
	db, err := database.NewClient()
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Create a new server instance
	s := &server{
		db:  db,
		log: log,
	}

	// Start the gRPC server
	go func() {
		if err := runGrpcServer(s); err != nil {
			log.Error("gRPC server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down servers...")
}

func runGrpcServer(s *server) error {
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, s)

	s.log.Info("gRPC server listening", "port", grpcPort)
	return grpcServer.Serve(lis)
}

// TODO: HTTP gateway removed temporarily because grpc-gateway generated handler
// (RegisterUserServiceHandlerFromEndpoint) is not produced by current proto build.
// Integrate protoc-gen-grpc-gateway and protoc-gen-openapiv2 plugins, then
// restore an HTTP server (port :8081) exposing REST and OpenAPI spec.
