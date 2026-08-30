package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestValidatePayload(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		wantErr bool
	}{
		{
			name:    "empty payload is valid",
			payload: []byte(""),
			wantErr: false,
		},
		{
			name:    "valid JSON object",
			payload: []byte(`{"event_id":"visit-123","created_at":"2026-08-28T12:00:00Z"}`),
			wantErr: false,
		},
		{
			name:    "valid arbitrary JSON",
			payload: []byte(`{"test":true,"count":10}`),
			wantErr: false,
		},
		{
			name:    "invalid plain text string",
			payload: []byte("this is corrupt non-json text"),
			wantErr: true,
		},
		{
			name:    "malformed JSON structure",
			payload: []byte(`{"event_id":`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildDLQEnvelope(t *testing.T) {
	msg := kafka.Message{
		Topic:     "visits.requested",
		Partition: 2,
		Offset:    1042,
		Key:       []byte("visit-key-99"),
		Value:     []byte(`{"invalid":`),
		Time:      time.Now().UTC(),
	}

	topic := "visits.requested"
	category := "database_failure"
	errMsg := "insert failed after 4 attempts (1 initial + 3 retries): connection refused"
	attempts := 4

	data, err := buildDLQEnvelope(topic, msg, category, errMsg, attempts)
	if err != nil {
		t.Fatalf("buildDLQEnvelope() error = %v", err)
	}

	var env deadLetterEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("failed to unmarshal generated envelope: %v", err)
	}

	if env.OriginalTopic != topic {
		t.Errorf("OriginalTopic = %q, want %q", env.OriginalTopic, topic)
	}
	if env.OriginalPartition != 2 {
		t.Errorf("OriginalPartition = %d, want 2", env.OriginalPartition)
	}
	if env.OriginalOffset != 1042 {
		t.Errorf("OriginalOffset = %d, want 1042", env.OriginalOffset)
	}
	if env.OriginalKey != "visit-key-99" {
		t.Errorf("OriginalKey = %q, want %q", env.OriginalKey, "visit-key-99")
	}
	if env.OriginalPayload != `{"invalid":` {
		t.Errorf("OriginalPayload = %q, want %q", env.OriginalPayload, `{"invalid":`)
	}
	if env.ErrorCategory != category {
		t.Errorf("ErrorCategory = %q, want %q", env.ErrorCategory, category)
	}
	if env.ErrorMessage != errMsg {
		t.Errorf("ErrorMessage = %q, want %q", env.ErrorMessage, errMsg)
	}
	if env.AttemptCount != 4 {
		t.Errorf("AttemptCount = %d, want 4", env.AttemptCount)
	}
	if env.FailedAt.IsZero() {
		t.Errorf("FailedAt should not be zero")
	}
}

func TestLoadConfig(t *testing.T) {
	os.Setenv("DB_PASSWORD", "secret")
	defer os.Unsetenv("DB_PASSWORD")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() failed: %v", err)
	}

	if cfg.kafkaTopic != "visits.requested" {
		t.Errorf("expected default topic 'visits.requested', got %q", cfg.kafkaTopic)
	}
	if cfg.kafkaDLQTopic != "visits.dead-letter" {
		t.Errorf("expected default DLQ topic 'visits.dead-letter', got %q", cfg.kafkaDLQTopic)
	}
	if cfg.maxRetries != 3 {
		t.Errorf("expected default maxRetries 3, got %d", cfg.maxRetries)
	}
	if cfg.ratePerSecond != 1 {
		t.Errorf("expected default ratePerSecond 1, got %d", cfg.ratePerSecond)
	}
}

func TestCreateTLSConfig(t *testing.T) {
	cfg := config{
		kafkaTLSEnabled: false,
	}
	tlsConfig, err := createTLSConfig(cfg)
	if err != nil {
		t.Fatalf("createTLSConfig(disabled) error: %v", err)
	}
	if tlsConfig != nil {
		t.Errorf("expected nil tlsConfig when disabled, got %+v", tlsConfig)
	}

	cfgErr := config{
		kafkaTLSEnabled:  true,
		kafkaTLSCAFile:   "/nonexistent/ca.crt",
		kafkaTLSCertFile: "/nonexistent/user.crt",
		kafkaTLSKeyFile:  "/nonexistent/user.key",
	}
	_, err = createTLSConfig(cfgErr)
	if err == nil {
		t.Errorf("expected error for nonexistent cert files, got nil")
	}
}

