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

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type createVisitEventResponse struct {
	Data struct {
		Status string `json:"status"`
		Queued int    `json:"queued"`
	} `json:"data"`
}

type visitCountResponse struct {
	Data struct {
		Count int64 `json:"count"`
	} `json:"data"`
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
		BatchSize:    1,
		BatchTimeout: 50 * time.Millisecond,
	}
	defer writer.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handleCreateVisitEvent := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		count, err := parseCount(r.URL.Query().Get("count"))
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_count", "count must be an integer between 1 and 100")
			return
		}

		for i := 0; i < count; i++ {
			if err := enqueueVisitEvent(r.Context(), cfg.requestTimeoutS, writer); err != nil {
				log.Printf("kafka publish failed: %v", err)
				writeAPIError(w, http.StatusServiceUnavailable, "queue_publish_failed", "queue publish failed")
				return
			}
		}

		response := createVisitEventResponse{}
		response.Data.Status = "queued"
		response.Data.Queued = count
		writeJSON(w, http.StatusAccepted, response)
	}

	handleVisitCount := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), cfg.requestTimeoutS*time.Second)
		defer cancel()

		var count int64
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM visits").Scan(&count); err != nil {
			log.Printf("count query failed: %v", err)
			writeAPIError(w, http.StatusServiceUnavailable, "db_query_failed", "db query failed")
			return
		}

		response := visitCountResponse{}
		response.Data.Count = count
		writeJSON(w, http.StatusOK, response)
	}

	mux.HandleFunc("/api/v1/visit-events", handleCreateVisitEvent)
	mux.HandleFunc("/api/v1/visits/count", handleVisitCount)

	// Backward-compatible routes for existing callers.
	mux.HandleFunc("/api/visits", handleCreateVisitEvent)
	mux.HandleFunc("/api/visits/count", handleVisitCount)

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

func parseCount(raw string) (int, error) {
	if raw == "" {
		return 1, nil
	}
	count, err := strconv.Atoi(raw)
	if err != nil || count < 1 || count > 100 {
		return 0, fmt.Errorf("invalid count")
	}
	return count, nil
}

func enqueueVisitEvent(parent context.Context, timeoutS time.Duration, writer *kafka.Writer) error {
	e := visitEvent{EventID: fmt.Sprintf("visit-%d", time.Now().UnixNano()), CreatedAt: time.Now().UTC()}
	payload, err := json.Marshal(e)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, timeoutS*time.Second)
	defer cancel()
	return writer.WriteMessages(ctx, kafka.Message{Key: []byte(e.EventID), Value: payload})
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]apiError{"error": {Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("json encode failed: %v", err)
	}
}
