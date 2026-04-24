package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func WaitKafkaEvent(
	t *testing.T,
	brokersCSV string,
	topic string,
	timeout time.Duration,
	match func(map[string]any) bool,
) bool {
	t.Helper()

	brokers := splitCSV(brokersCSV)
	if len(brokers) == 0 {
		return false
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     fmt.Sprintf("e2e-%s-%d", topic, time.Now().UnixNano()),
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    1_000_000,
		MaxWait:     500 * time.Millisecond,
	})
	defer reader.Close()

	var lastErr error
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
		msg, err := reader.ReadMessage(ctx)
		cancel()
		if err != nil {
			lastErr = err
			if strings.Contains(strings.ToLower(err.Error()), "context deadline") {
				continue
			}
			if strings.Contains(strings.ToLower(err.Error()), "unknown topic") {
				continue
			}
			continue
		}

		payload := map[string]any{}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			continue
		}
		if match(payload) {
			return true
		}
	}
	if lastErr != nil {
		t.Logf("kafka read timeout for topic=%s brokers=%v last_err=%v", topic, brokers, lastErr)
	}
	return false
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
