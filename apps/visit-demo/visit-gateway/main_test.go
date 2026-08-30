package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "empty defaults to 1", input: "", want: 1, wantErr: false},
		{name: "valid single digit", input: "5", want: 5, wantErr: false},
		{name: "valid max count", input: "100", want: 100, wantErr: false},
		{name: "invalid zero", input: "0", want: 0, wantErr: true},
		{name: "invalid negative", input: "-5", want: 0, wantErr: true},
		{name: "invalid over max", input: "101", want: 0, wantErr: true},
		{name: "invalid string", input: "abc", want: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCount(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseCount(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	os.Setenv("DB_PASSWORD", "secret")
	os.Setenv("KAFKA_TLS_ENABLED", "true")
	os.Setenv("KAFKA_BROKERS", "kafka:9093")
	defer func() {
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("KAFKA_TLS_ENABLED")
		os.Unsetenv("KAFKA_BROKERS")
	}()

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() failed: %v", err)
	}

	if !cfg.kafkaTLSEnabled {
		t.Errorf("expected kafkaTLSEnabled = true, got false")
	}
	if len(cfg.kafkaBrokers) != 1 || cfg.kafkaBrokers[0] != "kafka:9093" {
		t.Errorf("expected broker kafka:9093, got %v", cfg.kafkaBrokers)
	}
}

func TestCreateTLSConfig(t *testing.T) {
	// 1. When TLS is disabled and no cert paths are given, returns nil
	cfg := config{
		kafkaTLSEnabled: false,
	}
	tlsConfig, err := createTLSConfig(cfg)
	if err != nil {
		t.Fatalf("createTLSConfig(disabled) unexpected error: %v", err)
	}
	if tlsConfig != nil {
		t.Errorf("expected nil tlsConfig when disabled, got %+v", tlsConfig)
	}

	// 2. When enabled with invalid cert path, returns error
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

	// 3. When enabled with valid mock PEM files
	tempDir := t.TempDir()
	caFile := filepath.Join(tempDir, "ca.crt")
	_ = os.WriteFile(caFile, []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"), 0600)

	cfgCA := config{
		kafkaTLSEnabled: true,
		kafkaTLSCAFile:  caFile,
	}
	_, err = createTLSConfig(cfgCA)
	if err == nil {
		t.Errorf("expected error parsing invalid PEM content, got nil")
	}
}
