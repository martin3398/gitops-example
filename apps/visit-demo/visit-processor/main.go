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
	kafkaDLQTopic   string
	kafkaGroupID    string
	dbHost          string
	dbPort          string
	dbName          string
	dbUser          string
	dbPassword      string
	ratePerSecond   int
	maxRetries      int
	retryBackoffMs  int
	requestTimeoutS time.Duration
}

type deadLetterEnvelope struct {
	OriginalTopic     string    `json:"original_topic"`
	OriginalPartition int       `json:"original_partition"`
	OriginalOffset    int64     `json:"original_offset"`
	OriginalKey       string    `json:"original_key"`
	OriginalPayload   string    `json:"original_payload"`
	ErrorMessage      string    `json:"error_message"`
	ErrorCategory     string    `json:"error_category"`
	FailedAt          time.Time `json:"failed_at"`
	AttemptCount      int       `json:"attempt_count"`
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

	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.kafkaBrokers...),
		Topic:        cfg.kafkaDLQTopic,
		RequiredAcks: kafka.RequireAll,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,
		BatchTimeout: 50 * time.Millisecond,
	}
	defer dlqWriter.Close()

	go startHealthEndpoint(cfg.listenAddr)

	interval := time.Second / time.Duration(cfg.ratePerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("visit-processor started; topic=%s dlq=%s per-pod rate limit=%d op/sec retries=%d (total attempts=%d)",
		cfg.kafkaTopic, cfg.kafkaDLQTopic, cfg.ratePerSecond, cfg.maxRetries, 1+cfg.maxRetries)

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

		// 1. Validate payload format (detect poison pills)
		if err := validatePayload(msg.Value); err != nil {
			log.Printf("corrupt payload at offset %d, routing to DLQ: %v", msg.Offset, err)
			ctxDLQ, cancelDLQ := context.WithTimeout(context.Background(), cfg.requestTimeoutS*time.Second)
			dlqErr := sendToDLQ(ctxDLQ, dlqWriter, cfg.kafkaTopic, msg, "corrupt_payload", err.Error(), 1)
			cancelDLQ()
			if dlqErr != nil {
				log.Printf("CRITICAL: failed to produce corrupt message to DLQ (offset %d): %v; offset NOT committed", msg.Offset, dlqErr)
				continue
			}

			// Successfully diverted to DLQ, commit original offset so consumer group does not stall
			ctxCommit, cancelCommit := context.WithTimeout(context.Background(), cfg.requestTimeoutS*time.Second)
			commitErr := reader.CommitMessages(ctxCommit, msg)
			cancelCommit()
			if commitErr != nil {
				log.Printf("commit failed after DLQ routing (offset %d): %v", msg.Offset, commitErr)
			} else {
				log.Printf("diverted corrupt message (offset %d) to DLQ topic %s and committed offset", msg.Offset, cfg.kafkaDLQTopic)
			}
			continue
		}

		// 2. Process message into database with retries inside a PostgreSQL transaction (sql.Tx)
		attempts, err := insertVisitWithRetry(db, cfg.maxRetries, time.Duration(cfg.retryBackoffMs)*time.Millisecond)
		if err != nil {
			log.Printf("database transaction failed after %d attempts (offset %d), routing to DLQ: %v", attempts, msg.Offset, err)
			ctxDLQ, cancelDLQ := context.WithTimeout(context.Background(), cfg.requestTimeoutS*time.Second)
			dlqErr := sendToDLQ(ctxDLQ, dlqWriter, cfg.kafkaTopic, msg, "database_failure", err.Error(), attempts)
			cancelDLQ()
			if dlqErr != nil {
				log.Printf("CRITICAL: failed to produce DB failure message to DLQ (offset %d): %v; offset NOT committed", msg.Offset, dlqErr)
				continue
			}

			ctxCommit, cancelCommit := context.WithTimeout(context.Background(), cfg.requestTimeoutS*time.Second)
			commitErr := reader.CommitMessages(ctxCommit, msg)
			cancelCommit()
			if commitErr != nil {
				log.Printf("commit failed after DLQ routing (offset %d): %v", msg.Offset, commitErr)
			} else {
				log.Printf("diverted failed message (offset %d) to DLQ topic %s after %d attempts and committed offset", msg.Offset, cfg.kafkaDLQTopic, attempts)
			}
			continue
		}

		// 3. Normal commit path
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

	maxRetries, err := strconv.Atoi(envOrDefault("MAX_RETRIES", "3"))
	if err != nil || maxRetries < 0 {
		return config{}, fmt.Errorf("MAX_RETRIES must be >=0")
	}

	retryBackoffMs, err := strconv.Atoi(envOrDefault("RETRY_BACKOFF_MS", "100"))
	if err != nil || retryBackoffMs < 0 {
		return config{}, fmt.Errorf("RETRY_BACKOFF_MS must be >=0")
	}

	cfg := config{
		listenAddr:      envOrDefault("LISTEN_ADDR", ":8080"),
		kafkaTopic:      envOrDefault("KAFKA_TOPIC", "visits.requested"),
		kafkaDLQTopic:   envOrDefault("KAFKA_DLQ_TOPIC", "visits.dead-letter"),
		kafkaGroupID:    envOrDefault("KAFKA_GROUP_ID", "visit-processor-v1"),
		dbHost:          envOrDefault("DB_HOST", "postgres-rw.data-postgres.svc"),
		dbPort:          envOrDefault("DB_PORT", "5432"),
		dbName:          envOrDefault("DB_NAME", "app"),
		dbUser:          envOrDefault("DB_USER", "app"),
		dbPassword:      os.Getenv("DB_PASSWORD"),
		ratePerSecond:   rate,
		maxRetries:      maxRetries,
		retryBackoffMs:  retryBackoffMs,
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

func validatePayload(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	var raw any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}

func insertVisitTx(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin db transaction failed: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, "INSERT INTO visits (created_at) VALUES (NOW())"); err != nil {
		return fmt.Errorf("insert query failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit db transaction failed: %w", err)
	}
	return nil
}

func insertVisitWithRetry(db *sql.DB, maxRetries int, backoff time.Duration) (int, error) {
	totalAttempts := 1 + maxRetries
	var lastErr error
	for attempt := 1; attempt <= totalAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := insertVisitTx(ctx, db)
		cancel()
		if err == nil {
			return attempt, nil
		}
		lastErr = err
		if attempt < totalAttempts && backoff > 0 {
			time.Sleep(backoff * time.Duration(attempt))
		}
	}
	return totalAttempts, fmt.Errorf("insert failed after %d attempts (1 initial + %d retries): %w", totalAttempts, maxRetries, lastErr)
}

func buildDLQEnvelope(topic string, msg kafka.Message, category, errMsg string, attempts int) ([]byte, error) {
	envelope := deadLetterEnvelope{
		OriginalTopic:     topic,
		OriginalPartition: msg.Partition,
		OriginalOffset:    msg.Offset,
		OriginalKey:       string(msg.Key),
		OriginalPayload:   string(msg.Value),
		ErrorMessage:      errMsg,
		ErrorCategory:     category,
		FailedAt:          time.Now().UTC(),
		AttemptCount:      attempts,
	}
	return json.Marshal(envelope)
}

func sendToDLQ(ctx context.Context, writer *kafka.Writer, topic string, msg kafka.Message, category, errMsg string, attempts int) error {
	dlqPayload, err := buildDLQEnvelope(topic, msg, category, errMsg, attempts)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ envelope: %w", err)
	}
	key := msg.Key
	if len(key) == 0 {
		key = []byte(fmt.Sprintf("dlq-%d-%d", msg.Partition, msg.Offset))
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: dlqPayload,
	})
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
