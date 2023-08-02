package quickiedata

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type HTTPClientSettings struct {
	// Headers that will be added to every request
	// Include headers like "User-Agent" and "Authorization" here
	DefaultHeaders http.Header
	// Minimum time between successful requests
	RequestInterval time.Duration
	// Amount of time to wait between errors
	// Doubled after every successive fail
	Backoff time.Duration
	// Maximum amount of time to wait between errors
	// Backoff will be clamped to this value
	MaxBackoff time.Duration
	// Number of times to retry the request on failure
	MaxRetries int
	// Maximum number of connections per host
	MaxConnsPerHost int
}

func QuickieHTTPClient(settings *HTTPClientSettings) *http.Client {
	// Set up a new HTTP client with a custom transport that enforces a rate limit
	limiter := rate.NewLimiter(rate.Every(settings.RequestInterval), 1)

	// give some sensible default values
	maxRetries := settings.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}

	backoff := settings.Backoff
	if backoff <= 0 {
		backoff = 1 * time.Second
	}

	maxBackoff := settings.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = settings.MaxConnsPerHost

	return &http.Client{
		Transport: &quickieRoundTripper{
			roundTripper:   transport,
			limiter:        limiter,
			defaultHeaders: settings.DefaultHeaders,
			backoff:        backoff,
			maxBackoff:     maxBackoff,
			maxRetries:     maxRetries,
		},
	}
}

// quickieRoundTripper is a custom transport that enforces a rate limit on requests
type quickieRoundTripper struct {
	roundTripper   http.RoundTripper // underlying RoundTripper to use
	limiter        *rate.Limiter
	defaultHeaders http.Header
	backoff        time.Duration
	maxBackoff     time.Duration
	maxRetries     int
}

// Custom RoundTrip function that implements retries, backoff, user agent, handles errors,
func (rt *quickieRoundTripper) RoundTrip(origReq *http.Request) (*http.Response, error) {
	// set the initial backoff
	backoff := rt.backoff
	deadline, deadlineSet := origReq.Context().Deadline()

	for retries := 0; retries <= rt.maxRetries; retries++ {
		// Check if the original context is done on the original request before sending
		select {
		case <-origReq.Context().Done():
			return nil, origReq.Context().Err()
		default:
		}

		// Need to clone the request each time the request body is reset
		var req *http.Request
		if deadlineSet {
			ctx, cancel := context.WithTimeout(context.Background(), time.Until(deadline))
			defer cancel()
			req = origReq.Clone(ctx)
		} else {
			req = origReq.Clone(context.Background())
		}

		// Wait for the limiter before sending the request
		if err := rt.limiter.Wait(req.Context()); err != nil {
			return nil, err
		}

		// Set default headers (used for User-Agent and Authorization)
		if len(rt.defaultHeaders) > 0 {
			for key, values := range rt.defaultHeaders {
				if len(values) == 0 {
					continue
				}
				// Support header keys with multiple values
				req.Header.Del(key)
				for _, val := range values {
					req.Header.Add(key, val)
				}
			}
		}

		// Send the request
		resp, err := rt.roundTripper.RoundTrip(req)

		// On the last retry, return the result no matter way
		if retries >= rt.maxRetries {
			return resp, err
		}

		// We might retry at this point
		if err == nil {
			switch resp.StatusCode {
			case 429, 503, 504:
				// too many requests, service unavailable, gateway timeout
				// the request is retryable, with backoff cause we're nice people
			default:
				// the request either was successful, or had an unretryable error
				// either way, send the result
				return resp, err
			}
		}

		// We are going to retry at this point
		if err != nil {
			// a network error has occurred, immediately retry
			// covers things like connection timeouts, dns resolution errors, connection reset
			// try again without additional sleep
			continue
		}

		// At this point we are left with retryable http status codes
		// 429 Too Many Requests
		// 503 Service Unavailable
		// 504 Gateway Timeout
		waitTime := rt.calculateWaitAfterError(resp, backoff)
		resp.Body.Close()

		select {
		case <-req.Context().Done(): // context cancelled
		case <-time.After(waitTime): // sleep
		}

		// double the backoff each retry
		backoff = backoff * 2
		if backoff > rt.maxBackoff {
			backoff = rt.maxBackoff
		}
	}

	return nil, fmt.Errorf("too many retries")
}

// Calculate a wait duration from Retry-After response header and current backoff
func (rt *quickieRoundTripper) calculateWaitAfterError(resp *http.Response, backoff time.Duration) time.Duration {
	waitTime := backoff
	// retry-after header sometimes set on 429 too many requests
	retryAfterHeader := resp.Header.Get("Retry-After")
	retryAfterDuration, err := time.ParseDuration(retryAfterHeader + "s")
	if err == nil {
		waitTime = time.Duration(retryAfterDuration)
	}

	if waitTime < backoff {
		waitTime = backoff
	}

	return waitTime
}
