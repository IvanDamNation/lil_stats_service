package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/IvanDamNation/lil_stats_service/internal/handler"
	"github.com/IvanDamNation/lil_stats_service/internal/storage"
	"github.com/IvanDamNation/lil_stats_service/pkg/env"
)

func main() {
	_ = env.LoadEnv(".env")

	host := env.GetEnv("SERVER_ADDRESS", "127.0.0.1")
	port := env.GetEnvInt("SERVER_PORT", 8080)
	addr := fmt.Sprintf("%s:%d", host, port)

	rt := env.GetEnvDuration("SERVER_TIMEOUT_READ", 5)
	wt := env.GetEnvDuration("SERVER_TIMEOUT_WRITE", 10)
	it := env.GetEnvDuration("SERVER_TIMEOUT_IDLE", 120)

	s := storage.NewStorage(storage.NowFunc)
	defer s.Stop()

	h := handler.NewHandler(s)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/click", h.Click)
	mux.HandleFunc("POST /api/v1/click_stats", h.YesterdayUniqueClicks)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  rt,
		WriteTimeout: wt,
		IdleTimeout:  it,
	}

	go func() {
		log.Printf("Starting server on %s...\n", addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}

	fmt.Println("server stopped")
}
