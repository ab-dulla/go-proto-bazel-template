package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	// Start the HTTP server (minimal gateway substitute)
	httpServer := newHTTPServer(log)
	go func() {
		log.Info("HTTP server listening", "port", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down servers...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP shutdown error", "error", err)
	}
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

// newHTTPServer creates a minimal HTTP server that proxies selected REST
// endpoints to the local gRPC server without grpc-gateway codegen. This is a
// temporary solution until protoc-gen-grpc-gateway and OpenAPI generation are
// integrated into the build.
func newHTTPServer(log *slog.Logger) *http.Server {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})

	// GET /v1/users/{id}
	mux.HandleFunc("/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Extract ID from path.
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/users/"), "/")
		id := parts[0]
		if id == "" {
			http.Error(w, "missing user id", http.StatusBadRequest)
			return
		}

		// Create a gRPC client connection.
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, "127.0.0.1"+grpcPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Error("failed to dial gRPC", "error", err)
			http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
			return
		}
		defer conn.Close()
		client := userpb.NewUserServiceClient(conn)
		user, err := client.GetUser(ctx, &userpb.GetUserRequest{UserId: id})
		if err != nil {
			log.Error("gRPC GetUser error", "error", err, "user_id", id)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Manual JSON to avoid adding dependencies; struct tags from proto ensure fields.
		fmt.Fprintf(w, `{"userId":"%s","displayName":"%s","email":"%s"}`, user.UserId, user.DisplayName, user.Email)
	})

	return &http.Server{
		Addr:    httpPort,
		Handler: mux,
	}
}

// TODO: Replace manual HTTP shim with grpc-gateway generated handlers and real
// OpenAPI spec once protoc-gen-grpc-gateway & protoc-gen-openapiv2 are added.
