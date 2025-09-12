package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// Server represents the HTTP server for the API
type Server struct {
	addr     string
	handler  *Handler
	server   *http.Server
	listener net.Listener
}

// NewServer creates a new HTTP server
func NewServer(addr string, orchestrator Orchestrator) *Server {
	handler := NewHandler(orchestrator)
	
	mux := http.NewServeMux()
	
	// Add health endpoint
	mux.HandleFunc("/health", healthHandler)
	
	// Add API routes with middleware
	mux.Handle("/", withMiddleware(handler))
	
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return &Server{
		addr:    addr,
		handler: handler,
		server:  server,
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Create listener
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	
	s.listener = listener
	
	// Update addr with actual address (important for random ports)
	s.addr = listener.Addr().String()
	
	log.Printf("Starting API server on %s", s.addr)
	
	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()
	
	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Shutting down API server...")
		
		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		// Graceful shutdown
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
			return err
		}
		
		log.Println("API server stopped")
		return nil
		
	case err := <-errCh:
		return err
	}
}

// GetAddr returns the server address in a format suitable for HTTP clients
func (s *Server) GetAddr() string {
	if s.listener != nil {
		addr := s.listener.Addr().String()
		// Handle IPv6 addresses by extracting just the port
		if host, port, err := net.SplitHostPort(addr); err == nil {
			// If it's IPv6 or unspecified, use localhost
			if host == "::" || host == "" {
				return ":" + port
			}
			return net.JoinHostPort(host, port)
		}
	}
	return s.addr
}

// healthHandler handles health check requests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "dubernetes-api",
	}
	
	json.NewEncoder(w).Encode(response)
}

// withMiddleware wraps a handler with middleware
func withMiddleware(next http.Handler) http.Handler {
	return corsMiddleware(loggingMiddleware(next))
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Call next handler
		next.ServeHTTP(wrapped, r)
		
		// Log request
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}