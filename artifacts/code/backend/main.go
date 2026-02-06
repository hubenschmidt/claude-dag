package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	store := NewTaskStore()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HandleHealth)
	mux.HandleFunc("GET /tasks", HandleListTasks(store))
	mux.HandleFunc("POST /tasks", HandleCreateTask(store))
	mux.HandleFunc("GET /tasks/{id}", HandleGetTask(store))
	mux.HandleFunc("PUT /tasks/{id}", HandleUpdateTask(store))
	mux.HandleFunc("DELETE /tasks/{id}", HandleDeleteTask(store))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: corsMiddleware(mux),
	}

	go func() {
		log.Println("server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}
