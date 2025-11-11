package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	"go-monorepo-template/pkg/database"
	"go-monorepo-template/pkg/logger"
	userpb "go-monorepo-template/proto/acme/user/v1"
)

const (
	grpcPort = ":8080"
	httpPort = ":8081"
)

// server implements the UserService gRPC server.
type server struct {
	userpb.UnimplementedUserServiceServer
	db  *gorm.DB
	log *slog.Logger
}

func (s *server) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.User, error) {
	s.log.Info("GetUser called", "user_id", req.UserId)
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

	// Start the gRPC-Gateway HTTP server
	go func() {
		if err := runHttpServer(); err != nil {
			log.Error("HTTP server failed", "error", err)
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

func runHttpServer() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := userpb.RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost"+grpcPort, opts)
	if err != nil {
		return fmt.Errorf("failed to register gRPC gateway: %w", err)
	}

	// Serve the OpenAPI spec
	mux.HandlePath("GET", "/openapi.json", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.ServeFile(w, r, "services/user-api/user_v1.swagger.json")
	})

	s := &http.Server{
		Addr:    httpPort,
		Handler: mux,
	}

	fmt.Println("HTTP server listening on port", httpPort)
	return s.ListenAndServe()
}
