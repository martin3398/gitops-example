package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/segmentio/kafka-go"
)

type config struct {
	listenAddr      string
	kafkaBrokers    []string
	kafkaTopic      string
	kafkaGroupID    string
	dbHost          string
	dbPort          string
	dbName          string
	dbUser          string
	dbPassword      string
	ratePerSecond   int
	requestTimeoutS time.Duration
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db := openDB(cfg)
	defer db.Close()

	if err := ensureVisitsTable(db); err != nil {
		log.Fatalf("db init error: %v", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.kafkaBrokers,
		GroupID:     cfg.kafkaGroupID,
		Topic:       cfg.kafkaTopic,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	go startHealthEndpoint(cfg.listenAddr)

	interval := time.Second / time.Duration(cfg.ratePerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("visit-processor started; per-pod rate limit=%d op/sec", cfg.ratePerSecond)
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.requestTimeoutS*time.Second)
		msg, err := reader.FetchMessage(ctx)
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			log.Printf("fetch message failed: %v", err)
			continue
		}

		if err := handleMessage(db, msg.Value); err != nil {
			log.Printf("process failed (offset %d): %v", msg.Offset, err)
			continue
		}

		ctxCommit, cancelCommit := context.WithTimeout(context.Background(), cfg.requestTimeoutS*time.Second)
		err = reader.CommitMessages(ctxCommit, msg)
		cancelCommit()
		if err != nil {
			log.Printf("commit failed (offset %d): %v", msg.Offset, err)
			continue
		}

		log.Printf("processed visit event at offset %d", msg.Offset)
	}
}

func loadConfig() (config, error) {
	rate, err := strconv.Atoi(envOrDefault("RATE_LIMIT_PER_SEC", "1"))
	if err != nil || rate < 1 {
		return config{}, fmt.Errorf("RATE_LIMIT_PER_SEC must be >=1")
	}

	cfg := config{
		listenAddr:      envOrDefault("LISTEN_ADDR", ":8080"),
		kafkaTopic:      envOrDefault("KAFKA_TOPIC", "visits.requested"),
		kafkaGroupID:    envOrDefault("KAFKA_GROUP_ID", "visit-processor-v1"),
		dbHost:          envOrDefault("DB_HOST", "postgres-rw.data-postgres.svc"),
		dbPort:          envOrDefault("DB_PORT", "5432"),
		dbName:          envOrDefault("DB_NAME", "app"),
		dbUser:          envOrDefault("DB_USER", "app"),
		dbPassword:      os.Getenv("DB_PASSWORD"),
		ratePerSecond:   rate,
		requestTimeoutS: 5,
	}

	cfg.kafkaBrokers = splitCSV(envOrDefault("KAFKA_BROKERS", "kafka-kafka-bootstrap.data-kafka.svc:9092"))
	if len(cfg.kafkaBrokers) == 0 {
		return cfg, fmt.Errorf("KAFKA_BROKERS is required")
	}
	if cfg.dbPassword == "" {
		return cfg, fmt.Errorf("DB_PASSWORD is required")
	}
	return cfg, nil
}

func handleMessage(db *sql.DB, payload []byte) error {
	if len(payload) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(payload, &raw); err != nil {
			return fmt.Errorf("invalid message payload: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, "INSERT INTO visits (created_at) VALUES (NOW())")
	return err
}

func startHealthEndpoint(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("health server error: %v", err)
	}
}

func openDB(cfg config) *sql.DB {
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", cfg.dbHost, cfg.dbPort, cfg.dbName, cfg.dbUser, cfg.dbPassword)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db failed: %v", err)
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	return db
}

func ensureVisitsTable(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS visits (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
	return err
}

func envOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
