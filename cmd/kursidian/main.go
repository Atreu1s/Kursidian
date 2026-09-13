package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kursidian/internal/db"
	"kursidian/internal/handlers"
	"kursidian/internal/queries"
	"kursidian/web"
)

func main() {
	address := flag.String("addr", ":8080", "адрес, на котором слушает веб-сервер")
	databasePath := flag.String("db", "app.db", "путь к файлу базы данных SQLite")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(*address, *databasePath, logger); err != nil {
		logger.Error("приложение остановлено с ошибкой", "error", err)
		os.Exit(1)
	}
}

func browserURL(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "http://" + address
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	return "http://" + net.JoinHostPort(host, port)
}

func run(address, databasePath string, logger *slog.Logger) error {
	handle, err := db.Open(databasePath)
	if err != nil {
		return err
	}
	defer handle.Close()

	if err := db.Migrate(handle); err != nil {
		return err
	}

	empty, err := db.IsEmpty(handle)
	if err != nil {
		return err
	}

	if empty {
		logger.Info("база пуста, добавляю тестовые данные", "database", databasePath)
		if err := db.Seed(handle); err != nil {
			return err
		}
	}

	store := queries.New(handle)

	handler, err := handlers.New(store, web.Files, logger)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              address,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownFinished := make(chan struct{})

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		<-signals

		logger.Info("получен сигнал остановки, завершаю работу")

		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownContext); err != nil {
			logger.Error("не удалось корректно остановить сервер", "error", err)
		}

		close(shutdownFinished)
	}()

	logger.Info("Kursidian запущен", "address", browserURL(address), "database", databasePath)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	<-shutdownFinished

	return nil
}
