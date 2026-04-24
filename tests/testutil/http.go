package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"testing"
	"time"
)

type Env struct {
	GatewayURL   string
	AuthURL      string
	ProjectURL   string
	LimitsURL    string
	CatalogURL   string
	AnalyticsURL string
	KafkaBrokers string
}

func LoadEnv() Env {
	return Env{
		GatewayURL:   env("GATEWAY_URL", "http://localhost:8080"),
		AuthURL:      env("AUTH_URL", "http://localhost:8081"),
		ProjectURL:   env("PROJECT_URL", "http://localhost:8082"),
		LimitsURL:    env("LIMITS_URL", "http://localhost:8083"),
		CatalogURL:   env("CATALOG_URL", "http://localhost:8084"),
		AnalyticsURL: env("ANALYTICS_URL", "http://localhost:8085"),
		KafkaBrokers: env("KAFKA_BROKERS", "localhost:19092"),
	}
}

func NewClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 25 * time.Second, Jar: jar}
}

func RequestJSON(t *testing.T, client *http.Client, method, url, token string, body any) (int, map[string]any, []byte) {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request failed: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed [%s %s]: %v", method, url, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	obj := map[string]any{}
	if len(raw) > 0 && strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		_ = json.Unmarshal(raw, &obj)
	}
	return resp.StatusCode, obj, raw
}

func RequireStatus(t *testing.T, got, want int, raw []byte) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected status: got=%d want=%d body=%s", got, want, string(raw))
	}
}

func MustString(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in response: %+v", key, m)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		t.Fatalf("invalid string value for %q: %#v", key, v)
	}
	return s
}

func RandomEmail(prefix string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s-%d-%d@example.local", prefix, time.Now().Unix(), r.Intn(100000))
}

func env(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}
