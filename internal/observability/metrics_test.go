package observability

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestMetricsWritePrometheusIncludesRuntimeAndIdentityCounters(t *testing.T) {
	metrics := NewMetrics()
	metrics.ObserveRequest("POST", "/auth/login", 200, 120*time.Millisecond)
	metrics.ObserveIdentityEvent("login", "success")
	metrics.ObserveRateLimitDecision("auth_login", "rejected")
	metrics.ObserveIdempotencyDecision("replayed")

	var buf bytes.Buffer
	if err := metrics.WritePrometheus(&buf, "test-version"); err != nil {
		t.Fatalf("WritePrometheus returned error: %v", err)
	}

	output := buf.String()
	expectedSnippets := []string{
		`taskflow_build_info{version="test-version"} 1`,
		`http_requests_total{method="POST",route="/auth/login",status="200"} 1`,
		`taskflow_identity_events_total{flow="login",outcome="success"} 1`,
		`taskflow_rate_limit_decisions_total{scope="auth_login",decision="rejected"} 1`,
		`taskflow_idempotency_decisions_total{decision="replayed"} 1`,
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expected metrics output to contain %q\nfull output:\n%s", snippet, output)
		}
	}
}
