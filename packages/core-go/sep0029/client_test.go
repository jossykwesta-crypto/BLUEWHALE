package sep0029

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const validTOML = "MEMO_REQUIRED_ACCOUNTS = [\"" + acctA + "\"]\n[DOCUMENTATION]\nORG_NAME = \"Example\"\n"

// flakyServer fails the first `failures` requests with status, then serves
// body. It counts every request it receives.
func flakyServer(t *testing.T, failures int32, status int, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n := calls.Add(1); n <= failures {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func fastClient(retries int) *Client {
	return NewClient(HTTPClientOptions{
		Timeout:         2 * time.Second,
		MaxRetries:      retries,
		BackoffDuration: time.Millisecond,
	})
}

func TestFetchRetriesTransientStatuses(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusTooManyRequests} {
		srv, calls := flakyServer(t, 2, status, validTOML)

		toml, err := fastClient(2).FetchTOMLFromURL(context.Background(), srv.URL)
		if err != nil {
			t.Fatalf("status %d: FetchTOMLFromURL: %v", status, err)
		}
		if toml.OrgName != "Example" {
			t.Fatalf("status %d: OrgName = %q", status, toml.OrgName)
		}
		if got := calls.Load(); got != 3 {
			t.Fatalf("status %d: %d attempts, want 3", status, got)
		}
	}
}

func TestFetchGivesUpAfterMaxRetries(t *testing.T) {
	srv, calls := flakyServer(t, 100, http.StatusServiceUnavailable, validTOML)

	_, err := fastClient(3).Fetch(context.Background(), srv.URL)

	var statusErr *StatusError
	if !errors.As(err, &statusErr) || !errors.Is(err, ErrUnexpectedStatus) {
		t.Fatalf("err = %v, want *StatusError", err)
	}
	if statusErr.StatusCode != http.StatusServiceUnavailable || statusErr.Attempts != 4 {
		t.Fatalf("StatusError = %+v, want 503 after 4 attempts", statusErr)
	}
	if got := calls.Load(); got != 4 {
		t.Fatalf("%d attempts, want 4", got)
	}
}

func TestFetchDoesNotRetryClientErrors(t *testing.T) {
	srv, calls := flakyServer(t, 100, http.StatusNotFound, validTOML)

	_, err := fastClient(3).Fetch(context.Background(), srv.URL)

	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err = %v, want 404 StatusError", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("%d attempts, want 1", got)
	}
}

func TestFetchNegativeMaxRetriesDisablesRetries(t *testing.T) {
	srv, calls := flakyServer(t, 1, http.StatusInternalServerError, validTOML)

	if _, err := fastClient(-1).Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("expected the single attempt to fail")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("%d attempts, want 1", got)
	}
}

func TestFetchRetriesAfterAttemptTimeout(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
			return
		}
		_, _ = w.Write([]byte(validTOML))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(HTTPClientOptions{Timeout: 50 * time.Millisecond, MaxRetries: 1, BackoffDuration: time.Millisecond})
	toml, err := c.FetchTOMLFromURL(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchTOMLFromURL: %v", err)
	}
	if toml.OrgName != "Example" || calls.Load() != 2 {
		t.Fatalf("OrgName = %q after %d attempts, want Example after 2", toml.OrgName, calls.Load())
	}
}

func TestFetchRejectsOversizedBody(t *testing.T) {
	for name, chunked := range map[string]bool{"content-length": false, "chunked": true} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body := strings.Repeat("x", MaxTOMLSize+1)
				if chunked {
					// Flushing before writing forces chunked encoding, so the
					// size is only discovered while reading.
					w.(http.Flusher).Flush()
				} else {
					w.Header().Set("Content-Length", strconv.Itoa(len(body)))
				}
				_, _ = w.Write([]byte(body))
			}))
			t.Cleanup(srv.Close)

			_, err := fastClient(2).Fetch(context.Background(), srv.URL)
			if !errors.Is(err, ErrTOMLTooLarge) {
				t.Fatalf("err = %v, want ErrTOMLTooLarge", err)
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("%d attempts, want 1 (oversize is not retried)", got)
			}
		})
	}
}

func TestFetchStopsWhenContextIsCancelled(t *testing.T) {
	srv, calls := flakyServer(t, 100, http.StatusServiceUnavailable, validTOML)
	c := NewClient(HTTPClientOptions{MaxRetries: 5, BackoffDuration: time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	_, err := c.Fetch(ctx, srv.URL)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("%d attempts, want 1", got)
	}
}

func TestFetchTOMLRejectsInvalidDomains(t *testing.T) {
	c := fastClient(0)
	for _, domain := range []string{"", "https://example.com", "example.com/path", "user@example.com", "exa mple.com", "example.com?x=1"} {
		if _, err := c.FetchTOML(context.Background(), domain); !errors.Is(err, ErrInvalidDomain) {
			t.Fatalf("FetchTOML(%q) err = %v, want ErrInvalidDomain", domain, err)
		}
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(HTTPClientOptions{})
	if c.timeout != DefaultTimeout || c.maxRetries != DefaultMaxRetries || c.backoff != DefaultBackoffDuration || c.http == nil {
		t.Fatalf("defaults not applied: %+v", c)
	}
}

func TestBackoffDoublesAndCaps(t *testing.T) {
	c := NewClient(HTTPClientOptions{BackoffDuration: time.Second})
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, maxBackoff, maxBackoff}
	for i, w := range want {
		if got := c.backoffFor(i + 1); got != w {
			t.Fatalf("backoffFor(%d) = %v, want %v", i+1, got, w)
		}
	}
}
