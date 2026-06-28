package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emityoffwhite/url-shortener/internal/config"
	"github.com/emityoffwhite/url-shortener/internal/handler"
	"github.com/emityoffwhite/url-shortener/internal/service"
	"github.com/emityoffwhite/url-shortener/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	store := storage.NewMemoryStorage()
	svc := service.NewURLService(store)
	h := handler.NewHandler(svc, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в отдельной горутине, чтобы основной поток
	// мог слушать сигналы ОС для graceful shutdown.
	go func() {
		logger.Info("starting server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Ждём SIGINT (Ctrl+C) или SIGTERM (отправляется Docker/Kubernetes при остановке контейнера).
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down server gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Shutdown дожидается завершения уже идущих запросов вместо того,
	// чтобы резко обрывать соединения - важно для продовых сервисов.
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
