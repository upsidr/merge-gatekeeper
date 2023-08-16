package github

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRoundTripSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rt := NewRetryTransport(1)
	rt.next = http.DefaultTransport

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	startTime := time.Now()
	resp, err := rt.RoundTrip(req)
	elapsedTime := time.Since(startTime)
	if err != nil {
		t.Fatalf("error during request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	minElapsedTime := time.Second
	if elapsedTime > minElapsedTime {
		t.Errorf("expected elapsed time <= %v, got %v", minElapsedTime, elapsedTime)
	}
}

func TestRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	retryCount := 2
	rt := NewRetryTransport(retryCount)
	rt.next = http.DefaultTransport

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	startTime := time.Now()
	resp, err := rt.RoundTrip(req)
	elapsedTime := time.Since(startTime)
	if err != nil {
		t.Fatalf("error during request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status code %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}

	minElapsedTime := time.Duration(math.Pow(2, float64(retryCount))) * time.Second
	if elapsedTime < minElapsedTime {
		t.Errorf("expected elapsed time >= %v, got %v", minElapsedTime, elapsedTime)
	}
}
