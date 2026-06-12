package observability

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

type requestMetric struct {
	Method      string
	Route       string
	Status      int
	Count       uint64
	DurationSum float64
}

type Metrics struct {
	mu        sync.RWMutex
	startedAt time.Time
	requests  map[string]*requestMetric
}

func NewMetrics() *Metrics {
	return &Metrics{
		startedAt: time.Now().UTC(),
		requests:  make(map[string]*requestMetric),
	}
}

func (m *Metrics) ObserveRequest(method, route string, status int, duration time.Duration) {
	if route == "" {
		route = "unmatched"
	}

	key := fmt.Sprintf("%s|%s|%d", method, route, status)

	m.mu.Lock()
	defer m.mu.Unlock()

	metric, ok := m.requests[key]
	if !ok {
		metric = &requestMetric{
			Method: method,
			Route:  route,
			Status: status,
		}
		m.requests[key] = metric
	}

	metric.Count++
	metric.DurationSum += duration.Seconds()
}

func (m *Metrics) WritePrometheus(w io.Writer, version string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, err := fmt.Fprintln(w, "# HELP taskflow_build_info Static build information."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE taskflow_build_info gauge"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "taskflow_build_info{version=%q} 1\n", version); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "# HELP taskflow_process_start_time_seconds Process start time in unix seconds."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE taskflow_process_start_time_seconds gauge"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "taskflow_process_start_time_seconds %.0f\n", float64(m.startedAt.Unix())); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "# HELP http_requests_total Total number of HTTP requests."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE http_requests_total counter"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "# HELP http_request_duration_seconds_sum Total HTTP request duration in seconds."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE http_request_duration_seconds_sum counter"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "# HELP http_request_duration_seconds_count Total HTTP request samples."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE http_request_duration_seconds_count counter"); err != nil {
		return err
	}

	metrics := make([]*requestMetric, 0, len(m.requests))
	for _, metric := range m.requests {
		metrics = append(metrics, metric)
	}
	sort.Slice(metrics, func(i, j int) bool {
		left := strings.Join([]string{metrics[i].Method, metrics[i].Route, fmt.Sprintf("%d", metrics[i].Status)}, "|")
		right := strings.Join([]string{metrics[j].Method, metrics[j].Route, fmt.Sprintf("%d", metrics[j].Status)}, "|")
		return left < right
	})

	for _, metric := range metrics {
		if _, err := fmt.Fprintf(
			w,
			"http_requests_total{method=%q,route=%q,status=%q} %d\n",
			metric.Method,
			metric.Route,
			fmt.Sprintf("%d", metric.Status),
			metric.Count,
		); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(
			w,
			"http_request_duration_seconds_sum{method=%q,route=%q,status=%q} %.6f\n",
			metric.Method,
			metric.Route,
			fmt.Sprintf("%d", metric.Status),
			metric.DurationSum,
		); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(
			w,
			"http_request_duration_seconds_count{method=%q,route=%q,status=%q} %d\n",
			metric.Method,
			metric.Route,
			fmt.Sprintf("%d", metric.Status),
			metric.Count,
		); err != nil {
			return err
		}
	}

	return nil
}
