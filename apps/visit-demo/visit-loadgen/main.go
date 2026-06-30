package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultBands = "idle=0:0.2:2,below=0.3:0.8:4,capacity=0.9:1.2:3,overload=2:3:2,burst=5:10:1"

type band struct {
	name   string
	minRPS float64
	maxRPS float64
	weight int
}

type config struct {
	baseURL        string
	mode           string
	fixedBand      string
	bands          []band
	minPhase       time.Duration
	maxPhase       time.Duration
	statusInterval time.Duration
	reloadInterval time.Duration
	requestTimeout time.Duration
	maxBatch       int
	seed           int64
	signature      string
}

type counters struct {
	sent   int64
	failed int64
}

type countResponse struct {
	Data struct {
		Count  int64  `json:"count"`
		Queued *int64 `json:"queued"`
		Queue  struct {
			Status string `json:"status"`
		} `json:"queue"`
	} `json:"data"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	rng := rand.New(rand.NewSource(cfg.seed))
	client := &http.Client{}
	stats := counters{}

	log.Printf("visit-loadgen started mode=%s fixed_band=%s target=%s bands=%s", cfg.mode, cfg.fixedBand, cfg.baseURL, cfg.signature)
	run(ctx, client, rng, cfg, &stats)
}

func run(ctx context.Context, client *http.Client, rng *rand.Rand, cfg config, stats *counters) {
	for ctx.Err() == nil {
		phaseBand, rate, duration := nextPhase(rng, cfg)
		phaseEnd := time.Now().Add(duration)
		log.Printf("phase start band=%s mode=%s rate=%.2f/s duration=%s", phaseBand.name, cfg.mode, rate, duration.Round(time.Second))

		ticker := time.NewTicker(200 * time.Millisecond)
		statusTicker := time.NewTicker(cfg.statusInterval)
		reloadTicker := time.NewTicker(cfg.reloadInterval)
		lastTick := time.Now()
		var accumulator float64
		phaseDone := false

		for !phaseDone && ctx.Err() == nil {
			select {
			case <-ctx.Done():
				phaseDone = true
			case now := <-ticker.C:
				elapsed := now.Sub(lastTick).Seconds()
				lastTick = now
				accumulator += rate * elapsed
				messages := int(math.Floor(accumulator))
				if messages > 0 {
					accumulator -= float64(messages)
					sendMessages(ctx, client, cfg, messages, stats)
				}
				if now.After(phaseEnd) {
					phaseDone = true
				}
			case <-statusTicker.C:
				logStatus(ctx, client, cfg, stats, phaseBand.name, rate, time.Until(phaseEnd))
			case <-reloadTicker.C:
				nextCfg, err := loadConfig()
				if err != nil {
					log.Printf("config reload failed: %v", err)
					continue
				}
				if nextCfg.signature != cfg.signature || nextCfg.mode != cfg.mode || nextCfg.fixedBand != cfg.fixedBand {
					log.Printf("config changed; ending current phase")
					cfg = nextCfg
					phaseDone = true
				}
			}
		}

		ticker.Stop()
		statusTicker.Stop()
		reloadTicker.Stop()
	}
}

func nextPhase(rng *rand.Rand, cfg config) (band, float64, time.Duration) {
	selected := cfg.bands[0]
	switch cfg.mode {
	case "paused":
		selected = band{name: "paused", minRPS: 0, maxRPS: 0, weight: 1}
	case "fixed":
		selected = cfg.bands[0]
		for _, candidate := range cfg.bands {
			if candidate.name == cfg.fixedBand {
				selected = candidate
				break
			}
		}
	default:
		selected = weightedBand(rng, cfg.bands)
	}

	rate := selected.minRPS
	if selected.maxRPS > selected.minRPS {
		rate += rng.Float64() * (selected.maxRPS - selected.minRPS)
	}
	duration := cfg.minPhase
	if cfg.maxPhase > cfg.minPhase {
		duration += time.Duration(rng.Int63n(int64(cfg.maxPhase - cfg.minPhase)))
	}
	return selected, rate, duration
}

func weightedBand(rng *rand.Rand, bands []band) band {
	total := 0
	for _, b := range bands {
		total += b.weight
	}
	if total <= 0 {
		return bands[0]
	}
	pick := rng.Intn(total)
	for _, b := range bands {
		if pick < b.weight {
			return b
		}
		pick -= b.weight
	}
	return bands[len(bands)-1]
}

func sendMessages(ctx context.Context, client *http.Client, cfg config, messages int, stats *counters) {
	for messages > 0 {
		batch := messages
		if batch > cfg.maxBatch {
			batch = cfg.maxBatch
		}
		messages -= batch

		requestCtx, cancel := context.WithTimeout(ctx, cfg.requestTimeout)
		endpoint := strings.TrimRight(cfg.baseURL, "/") + "/visit-events?count=" + strconv.Itoa(batch)
		req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, nil)
		if err != nil {
			cancel()
			stats.failed += int64(batch)
			log.Printf("build request failed: %v", err)
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			cancel()
			stats.failed += int64(batch)
			log.Printf("queue request failed count=%d: %v", batch, err)
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		cancel()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			stats.failed += int64(batch)
			log.Printf("queue request failed count=%d status=%d", batch, resp.StatusCode)
			continue
		}
		stats.sent += int64(batch)
	}
}

func logStatus(ctx context.Context, client *http.Client, cfg config, stats *counters, bandName string, rate float64, remaining time.Duration) {
	processed, queued, queueStatus, err := readCount(ctx, client, cfg)
	if err != nil {
		log.Printf("status band=%s rate=%.2f/s phase_remaining=%s sent=%d failed=%d count=unavailable queued=unavailable error=%v", bandName, rate, remaining.Round(time.Second), stats.sent, stats.failed, err)
		return
	}
	queuedText := "unavailable"
	if queued != nil && queueStatus == "ok" {
		queuedText = strconv.FormatInt(*queued, 10)
	}
	log.Printf("status band=%s rate=%.2f/s phase_remaining=%s sent=%d failed=%d count=%d queued=%s queue_status=%s", bandName, rate, remaining.Round(time.Second), stats.sent, stats.failed, processed, queuedText, queueStatus)
}

func readCount(ctx context.Context, client *http.Client, cfg config) (int64, *int64, string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, cfg.requestTimeout)
	defer cancel()
	endpoint := strings.TrimRight(cfg.baseURL, "/") + "/visits/count"
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, "unavailable", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, "unavailable", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return 0, nil, "unavailable", fmt.Errorf("count status %d", resp.StatusCode)
	}
	var payload countResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, nil, "unavailable", err
	}
	return payload.Data.Count, payload.Data.Queued, payload.Data.Queue.Status, nil
}

func loadConfig() (config, error) {
	values := map[string]string{
		"TARGET_BASE_URL":     envOrDefault("TARGET_BASE_URL", "http://visit-web-visit-gateway.visit-web.svc/api/v1"),
		"LOADGEN_MODE":        envOrDefault("LOADGEN_MODE", "paused"),
		"LOADGEN_FIXED_BAND":  envOrDefault("LOADGEN_FIXED_BAND", "idle"),
		"LOADGEN_BANDS":       envOrDefault("LOADGEN_BANDS", defaultBands),
		"MIN_PHASE_DURATION":  envOrDefault("MIN_PHASE_DURATION", "30s"),
		"MAX_PHASE_DURATION":  envOrDefault("MAX_PHASE_DURATION", "180s"),
		"STATUS_INTERVAL":     envOrDefault("STATUS_INTERVAL", "10s"),
		"RELOAD_INTERVAL":     envOrDefault("RELOAD_INTERVAL", "10s"),
		"REQUEST_TIMEOUT":     envOrDefault("REQUEST_TIMEOUT", "5s"),
		"MAX_BATCH":           envOrDefault("MAX_BATCH", "100"),
		"LOADGEN_SEED":        envOrDefault("LOADGEN_SEED", strconv.FormatInt(time.Now().UnixNano(), 10)),
		"LOADGEN_CONFIG_FILE": envOrDefault("LOADGEN_CONFIG_FILE", "/etc/visit-loadgen/config.env"),
	}

	if fileValues, err := readConfigFile(values["LOADGEN_CONFIG_FILE"]); err == nil {
		for k, v := range fileValues {
			values[k] = v
		}
	} else if !os.IsNotExist(err) {
		return config{}, err
	}

	mode := strings.ToLower(strings.TrimSpace(values["LOADGEN_MODE"]))
	if mode != "paused" && mode != "random" && mode != "fixed" {
		return config{}, fmt.Errorf("LOADGEN_MODE must be paused, random, or fixed")
	}
	parsedURL, err := url.Parse(values["TARGET_BASE_URL"])
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return config{}, fmt.Errorf("TARGET_BASE_URL must be an absolute URL")
	}
	bands, err := parseBands(values["LOADGEN_BANDS"])
	if err != nil {
		return config{}, err
	}
	minPhase, err := time.ParseDuration(values["MIN_PHASE_DURATION"])
	if err != nil {
		return config{}, fmt.Errorf("MIN_PHASE_DURATION: %w", err)
	}
	maxPhase, err := time.ParseDuration(values["MAX_PHASE_DURATION"])
	if err != nil {
		return config{}, fmt.Errorf("MAX_PHASE_DURATION: %w", err)
	}
	if minPhase <= 0 || maxPhase < minPhase {
		return config{}, fmt.Errorf("phase durations must be positive and MAX_PHASE_DURATION must be >= MIN_PHASE_DURATION")
	}
	statusInterval, err := time.ParseDuration(values["STATUS_INTERVAL"])
	if err != nil || statusInterval <= 0 {
		return config{}, fmt.Errorf("STATUS_INTERVAL must be positive")
	}
	reloadInterval, err := time.ParseDuration(values["RELOAD_INTERVAL"])
	if err != nil || reloadInterval <= 0 {
		return config{}, fmt.Errorf("RELOAD_INTERVAL must be positive")
	}
	requestTimeout, err := time.ParseDuration(values["REQUEST_TIMEOUT"])
	if err != nil || requestTimeout <= 0 {
		return config{}, fmt.Errorf("REQUEST_TIMEOUT must be positive")
	}
	maxBatch, err := strconv.Atoi(values["MAX_BATCH"])
	if err != nil || maxBatch < 1 || maxBatch > 100 {
		return config{}, fmt.Errorf("MAX_BATCH must be between 1 and 100")
	}
	seed, err := strconv.ParseInt(values["LOADGEN_SEED"], 10, 64)
	if err != nil {
		return config{}, fmt.Errorf("LOADGEN_SEED must be an integer")
	}

	return config{
		baseURL:        values["TARGET_BASE_URL"],
		mode:           mode,
		fixedBand:      values["LOADGEN_FIXED_BAND"],
		bands:          bands,
		minPhase:       minPhase,
		maxPhase:       maxPhase,
		statusInterval: statusInterval,
		reloadInterval: reloadInterval,
		requestTimeout: requestTimeout,
		maxBatch:       maxBatch,
		seed:           seed,
		signature:      fmt.Sprintf("mode=%s fixed=%s bands=%s min=%s max=%s", mode, values["LOADGEN_FIXED_BAND"], values["LOADGEN_BANDS"], minPhase, maxPhase),
	}, nil
}

func parseBands(raw string) ([]band, error) {
	parts := strings.Split(raw, ",")
	bands := make([]band, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		nameAndSpec := strings.SplitN(part, "=", 2)
		if len(nameAndSpec) != 2 || strings.TrimSpace(nameAndSpec[0]) == "" {
			return nil, fmt.Errorf("invalid band %q", part)
		}
		spec := strings.Split(nameAndSpec[1], ":")
		if len(spec) != 3 {
			return nil, fmt.Errorf("band %q must use name=min:max:weight", part)
		}
		minRPS, err := strconv.ParseFloat(spec[0], 64)
		if err != nil || minRPS < 0 {
			return nil, fmt.Errorf("band %q has invalid min rate", part)
		}
		maxRPS, err := strconv.ParseFloat(spec[1], 64)
		if err != nil || maxRPS < minRPS {
			return nil, fmt.Errorf("band %q has invalid max rate", part)
		}
		weight, err := strconv.Atoi(spec[2])
		if err != nil || weight < 1 {
			return nil, fmt.Errorf("band %q has invalid weight", part)
		}
		bands = append(bands, band{name: strings.TrimSpace(nameAndSpec[0]), minRPS: minRPS, maxRPS: maxRPS, weight: weight})
	}
	if len(bands) == 0 {
		return nil, fmt.Errorf("at least one load band is required")
	}
	return bands, nil
}

func readConfigFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid config line %q", line)
		}
		values[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), `"`)
	}
	return values, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
