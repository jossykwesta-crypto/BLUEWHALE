package sep0029

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Defaults applied by NewClient for zero-valued HTTPClientOptions fields.
const (
	DefaultTimeout         = 10 * time.Second
	DefaultMaxRetries      = 2
	DefaultBackoffDuration = 250 * time.Millisecond

	// maxBackoff caps the exponential backoff between attempts.
	maxBackoff = 10 * time.Second
)

var (
	// ErrInvalidDomain reports a domain that is empty or is not a bare host
	// name (it contains a scheme, path, query, credentials or whitespace).
	ErrInvalidDomain = errors.New("sep0029: invalid domain")

	// ErrUnexpectedStatus reports a non-200 response. The concrete error is
	// a *StatusError.
	ErrUnexpectedStatus = errors.New("sep0029: unexpected HTTP status")
)

// StatusError is returned when the server answers with a non-200 status
// after all retries. It matches ErrUnexpectedStatus with errors.Is.
type StatusError struct {
	StatusCode int
	Attempts   int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("sep0029: unexpected HTTP status %d after %d attempt(s)", e.StatusCode, e.Attempts)
}

// Is makes errors.Is(err, ErrUnexpectedStatus) true for every *StatusError.
func (e *StatusError) Is(target error) bool { return target == ErrUnexpectedStatus }

// HTTPClientOptions configures Client. Zero values select the defaults.
type HTTPClientOptions struct {
	// Timeout bounds each attempt, including reading the body.
	// Zero means DefaultTimeout.
	Timeout time.Duration

	// MaxRetries is how many times a failed attempt is retried, so a request
	// makes at most MaxRetries+1 attempts. Zero means DefaultMaxRetries; a
	// negative value disables retries.
	MaxRetries int

	// BackoffDuration is the wait before the first retry. Each later retry
	// waits twice as long, up to 10s. Zero means DefaultBackoffDuration.
	BackoffDuration time.Duration

	// HTTPClient sends the requests. Nil means a new http.Client. Its own
	// Timeout is left alone; Timeout above is applied per attempt.
	HTTPClient *http.Client
}

// Client fetches stellar.toml files with a per-attempt timeout, retries with
// exponential backoff, and a MaxTOMLSize cap on the response body.
type Client struct {
	timeout    time.Duration
	maxRetries int
	backoff    time.Duration
	http       *http.Client
}

// NewClient returns a Client configured by opts.
func NewClient(opts HTTPClientOptions) *Client {
	c := &Client{
		timeout:    opts.Timeout,
		maxRetries: opts.MaxRetries,
		backoff:    opts.BackoffDuration,
		http:       opts.HTTPClient,
	}
	if c.timeout <= 0 {
		c.timeout = DefaultTimeout
	}
	if c.maxRetries == 0 {
		c.maxRetries = DefaultMaxRetries
	} else if c.maxRetries < 0 {
		c.maxRetries = 0
	}
	if c.backoff <= 0 {
		c.backoff = DefaultBackoffDuration
	}
	if c.http == nil {
		c.http = &http.Client{}
	}
	return c
}

// FetchTOML downloads and parses https://<domain>/.well-known/stellar.toml.
func (c *Client) FetchTOML(ctx context.Context, domain string) (*StellarTOML, error) {
	if !validDomain(domain) {
		return nil, ErrInvalidDomain
	}
	return c.FetchTOMLFromURL(ctx, "https://"+domain+"/.well-known/stellar.toml")
}

// FetchTOMLFromURL downloads and parses the stellar.toml at url.
func (c *Client) FetchTOMLFromURL(ctx context.Context, url string) (*StellarTOML, error) {
	body, err := c.Fetch(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseTOML(body)
}

// Fetch GETs url and returns a body of at most MaxTOMLSize bytes.
//
// Network errors, timeouts, 429 and 5xx responses are retried; other
// statuses, an oversized body and a cancelled ctx are returned at once.
func (c *Client) Fetch(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, c.backoffFor(attempt)); err != nil {
				return nil, err
			}
		}

		body, retry, err := c.attempt(ctx, url, attempt+1)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry || ctx.Err() != nil {
			break
		}
	}
	return nil, lastErr
}

// backoffFor returns the wait before the given retry (1-based).
func (c *Client) backoffFor(retry int) time.Duration {
	d := c.backoff
	for i := 1; i < retry && d < maxBackoff; i++ {
		d *= 2
	}
	if d > maxBackoff {
		d = maxBackoff
	}
	return d
}

// attempt makes one request. retry reports whether a failure is transient.
func (c *Client) attempt(ctx context.Context, url string, n int) (body []byte, retry bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("sep0029: building request: %w", err)
	}
	req.Header.Set("Accept", "text/plain, application/toml, */*")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, transient(err), fmt.Errorf("sep0029: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Drain a little so the connection can be reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		retry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return nil, retry, &StatusError{StatusCode: resp.StatusCode, Attempts: n}
	}

	if resp.ContentLength > MaxTOMLSize {
		return nil, false, ErrTOMLTooLarge
	}
	body, err = io.ReadAll(io.LimitReader(resp.Body, MaxTOMLSize+1))
	if err != nil {
		return nil, transient(err), fmt.Errorf("sep0029: reading body: %w", err)
	}
	if len(body) > MaxTOMLSize {
		return nil, false, ErrTOMLTooLarge
	}
	return body, false, nil
}

// transient reports whether a transport error is worth retrying. A caller's
// cancellation is not; a timeout or connection failure is.
func transient(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// validDomain accepts a bare host name with an optional port.
func validDomain(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}
	return !strings.ContainsAny(domain, "/?#@\\ \t\r\n") && !strings.Contains(domain, "://")
}
