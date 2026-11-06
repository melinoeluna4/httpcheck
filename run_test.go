package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCheckSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}
	checked := check(t.Context(), client, server.URL, http.StatusOK, 0, 0)

	if !checked.OK {
		t.Fatalf("checked.OK = false, error = %q", checked.Error)
	}
	if checked.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", checked.Status)
	}
}

func TestCheckRetries(t *testing.T) {
	t.Parallel()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}
	checked := check(t.Context(), client, server.URL, http.StatusOK, 1, 0)

	if !checked.OK {
		t.Fatalf("checked.OK = false, error = %q, attempts = %d", checked.Error, checked.Attempts)
	}
	if checked.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", checked.Attempts)
	}
}

func TestCheckMismatch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}
	checked := check(t.Context(), client, server.URL, http.StatusOK, 0, 0)

	if checked.OK {
		t.Fatal("checked.OK = true, want false")
	}
	if checked.Status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", checked.Status)
	}
}

func TestCheckNetworkError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	url := server.URL
	server.Close()

	client := &http.Client{Timeout: time.Second}
	checked := check(t.Context(), client, url, http.StatusOK, 0, 0)

	if checked.OK {
		t.Fatal("checked.OK = true, want false")
	}
	if checked.Error == "" {
		t.Fatal("checked.Error is empty, want a network error")
	}
}

func TestRunJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := run([]string{"-json", server.URL}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit code = %d, want 0; stderr = %s", code, stderr.String())
	}

	var results []result
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("json.Unmarshal: %v; output = %s", err, stdout.String())
	}
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("results = %#v, want one OK result", results)
	}
}

func TestRunFailsOnMismatch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := run([]string{server.URL}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "FAIL") {
		t.Fatalf("stdout = %q, want FAIL", stdout.String())
	}
}

func TestRunRejectsInvalidExpect(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run([]string{"-expect", "99", "https://example.com"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "expect") {
		t.Fatalf("stderr = %q, want an expect error", stderr.String())
	}
}
