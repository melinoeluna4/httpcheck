package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const usage = `httpcheck checks HTTP endpoints against an expected status code.

Usage:
  httpcheck [flags] URL [URL...]

Flags:
  -expect int         expected HTTP status (default 200)
  -timeout duration   per-request timeout (default 5s)
  -retries int        retries per URL (default 0)
  -interval duration  delay between retries (default 500ms)
  -json               print a JSON report

Examples:
  httpcheck https://example.com
  httpcheck -expect 204 https://example.com/health
  httpcheck -retries 3 -timeout 2s https://api.example.com/health

Exit codes:
  0  every URL returned the expected status
  1  at least one URL failed
  2  usage error
`

type result struct {
	URL      string `json:"url"`
	Status   int    `json:"status"`
	Expected int    `json:"expected"`
	Latency  string `json:"latency"`
	Attempts int    `json:"attempts"`
	Error    string `json:"error,omitempty"`
	OK       bool   `json:"ok"`
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("httpcheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprint(stderr, usage)
	}

	expect := flags.Int("expect", http.StatusOK, "expected HTTP status")
	timeout := flags.Duration("timeout", 5*time.Second, "per-request timeout")
	retries := flags.Int("retries", 0, "retries per URL")
	interval := flags.Duration("interval", 500*time.Millisecond, "delay between retries")
	asJSON := flags.Bool("json", false, "print a JSON report")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}

		return 2
	}

	if flags.NArg() == 0 {
		fmt.Fprintln(stderr, "httpcheck: at least one URL is required")
		flags.Usage()
		return 2
	}

	if *expect < 100 || *expect > 599 {
		fmt.Fprintln(stderr, "httpcheck: expect must be a valid HTTP status between 100 and 599")
		return 2
	}

	if *timeout <= 0 || *retries < 0 || *interval < 0 {
		fmt.Fprintln(stderr, "httpcheck: timeout must be positive and retries/interval cannot be negative")
		return 2
	}

	client := &http.Client{Timeout: *timeout}
	results := make([]result, 0, flags.NArg())
	failed := false

	for _, target := range flags.Args() {
		checked := check(context.Background(), client, target, *expect, *retries, *interval)
		results = append(results, checked)

		if !checked.OK {
			failed = true
		}
	}

	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(results); err != nil {
			fmt.Fprintf(stderr, "httpcheck: %v\n", err)
			return 2
		}
	} else {
		printReport(stdout, results, *expect)
	}

	if failed {
		return 1
	}

	return 0
}

func check(
	ctx context.Context,
	client *http.Client,
	target string,
	expected int,
	retries int,
	interval time.Duration,
) result {
	checked := result{URL: target, Expected: expected}
	start := time.Now()

	for attempt := 1; attempt <= retries+1; attempt++ {
		checked.Attempts = attempt

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			checked.Error = err.Error()
			break
		}

		response, err := client.Do(request)
		if err != nil {
			checked.Error = err.Error()
			if attempt <= retries {
				time.Sleep(interval)
			}

			continue
		}

		checked.Status = response.StatusCode
		_, copyErr := io.Copy(io.Discard, response.Body)
		closeErr := response.Body.Close()

		if copyErr != nil || closeErr != nil {
			if copyErr != nil {
				checked.Error = copyErr.Error()
			} else {
				checked.Error = closeErr.Error()
			}

			if attempt <= retries {
				time.Sleep(interval)
			}

			continue
		}

		checked.Error = ""

		if response.StatusCode == expected {
			checked.OK = true
			break
		}

		if attempt <= retries {
			time.Sleep(interval)
		}
	}

	checked.Latency = time.Since(start).Round(time.Millisecond).String()

	return checked
}

func printReport(writer io.Writer, results []result, expected int) {
	fmt.Fprintf(writer, "httpcheck: %d URL(s), expecting %d\n", len(results), expected)

	for _, checked := range results {
		status := fmt.Sprintf("%d", checked.Status)
		if checked.Status == 0 {
			status = "-"
		}

		if checked.OK {
			fmt.Fprintf(writer, "  %s %s %s\n", checked.URL, status, checked.Latency)
			continue
		}

		details := []string{
			fmt.Sprintf("expected %d", checked.Expected),
			fmt.Sprintf("attempts %d", checked.Attempts),
		}
		if checked.Error != "" {
			details = append(details, "error: "+checked.Error)
		}

		fmt.Fprintf(writer, "  %s %s %s (%s)\n", checked.URL, status, checked.Latency, strings.Join(details, ", "))
	}

	if len(results) == 0 {
		fmt.Fprintln(writer, "OK")
		return
	}

	failed := false
	for _, checked := range results {
		if !checked.OK {
			failed = true
			break
		}
	}

	if failed {
		fmt.Fprintln(writer, "FAIL")
		return
	}

	fmt.Fprintln(writer, "OK")
}
