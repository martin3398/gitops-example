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
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/segmentio/kafka-go"
)

type config struct {
	listenAddr      string
	kafkaBrokers    []string
	kafkaTopic      string
	dbHost          string
	dbPort          string
	dbName          string
	dbUser          string
	dbPassword      string
	requestTimeoutS time.Duration
}

type visitEvent struct {
	EventID   string    `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`
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

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.kafkaBrokers...),
		Topic:        cfg.kafkaTopic,
		RequiredAcks: kafka.RequireAll,
		Balancer:     &kafka.LeastBytes{},
	}
	defer writer.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/visits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		e := visitEvent{EventID: fmt.Sprintf("visit-%d", time.Now().UnixNano()), CreatedAt: time.Now().UTC()}
		payload, err := json.Marshal(e)
		if err != nil {
			http.Error(w, "encode failed", http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cfg.requestTimeoutS*time.Second)
		defer cancel()

		if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte(e.EventID), Value: payload}); err != nil {
			log.Printf("kafka publish failed: %v", err)
			http.Error(w, "queue publish failed", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "queued", "event_id": e.EventID})
	})

	mux.HandleFunc("/api/visits/count", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cfg.requestTimeoutS*time.Second)
		defer cancel()

		var count int64
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM visits").Scan(&count); err != nil {
			log.Printf("count query failed: %v", err)
			http.Error(w, "db query failed", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": count})
	})

	server := &http.Server{Addr: cfg.listenAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("visit-gateway listening on %s", cfg.listenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

func loadConfig() (config, error) {
	cfg := config{
		listenAddr:      envOrDefault("LISTEN_ADDR", ":8080"),
		kafkaTopic:      envOrDefault("KAFKA_TOPIC", "visits.requested"),
		dbHost:          envOrDefault("DB_HOST", "postgres-rw.data-postgres.svc"),
		dbPort:          envOrDefault("DB_PORT", "5432"),
		dbName:          envOrDefault("DB_NAME", "app"),
		dbUser:          envOrDefault("DB_USER", "app"),
		dbPassword:      os.Getenv("DB_PASSWORD"),
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
