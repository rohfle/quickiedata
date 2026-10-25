package quickiedata_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/rohfle/quickiedata"
)

// newUAServer serves both request kinds and records the User-Agent header of
// every request it receives.
func newUAServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()

	var mu sync.Mutex
	var seen []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("User-Agent"))
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, `{"head":{"vars":[]},"results":{"bindings":[]}}`)
			return
		}
		fmt.Fprint(w, `{"entities":{"Q1":{"id":"Q1","type":"item"}},"success":1}`)
	}))
	t.Cleanup(server.Close)

	return server, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

// TestUserAgentOnEveryRequest checks that both request kinds carry the caller's
// user agent with the library token appended.
func TestUserAgentOnEveryRequest(t *testing.T) {
	server, seen := newUAServer(t)
	const callerUA = "App/1.0 (mailto:app@example.org)"
	client := newClientForServer(t, callerUA, server)

	if _, err := client.GetEntities(context.Background(), []string{"Q1"}, nil); err != nil {
		t.Fatalf("GetEntities: %s", err)
	}
	query := quickiedata.NewSPARQLQuery()
	query.Template = "SELECT ?item WHERE { ?item wdt:P31 wd:Q146 } LIMIT 1"
	if _, err := client.SPARQLQueryRaw(context.Background(), query, nil); err != nil {
		t.Fatalf("SPARQLQueryRaw: %s", err)
	}

	got := seen()
	if len(got) != 2 {
		t.Fatalf("server saw %d requests, want 2", len(got))
	}
	want := callerUA + " " + quickiedata.LibraryUserAgent
	for i, userAgent := range got {
		if userAgent != want {
			t.Errorf("request %d user agent = %q, want %q", i+1, userAgent, want)
		}
	}
}

// TestLibraryTokenIsAddedOnce checks that a caller naming the library, or
// opting out of the token, does not get it twice.
func TestLibraryTokenIsAddedOnce(t *testing.T) {
	tests := []struct {
		name       string
		callerUA   string
		suppress   bool
		wantSuffix string
	}{
		{
			name:     "caller pinned the library version",
			callerUA: "App/1.0 quickiedata/v9.9",
		},
		{
			name:     "caller suppresses the library token",
			callerUA: "App/1.0 (mailto:app@example.org)",
			suppress: true,
		},
		{
			// Different application, despite the similar name
			name:       "cli user agent gets the token",
			callerUA:   "quickiedata-cli/0.1 (https://github.com/rohfle/quickiedata)",
			wantSuffix: " " + quickiedata.LibraryUserAgent,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, seen := newUAServer(t)
			client, err := quickiedata.NewClient(test.callerUA, &quickiedata.WikidataClientOptions{
				SuppressLibraryUA: test.suppress,
			})
			if err != nil {
				t.Fatalf("NewClient: %s", err)
			}
			client.APIEndpoint = server.URL

			// The response body does not matter here, only the header that
			// arrived with the request.
			_, _ = client.GetEntities(context.Background(), []string{"Q1"}, nil)

			got := seen()
			if len(got) != 1 {
				t.Fatalf("server saw %d requests, want 1", len(got))
			}
			want := test.callerUA + test.wantSuffix
			if got[0] != want {
				t.Errorf("user agent = %q, want %q", got[0], want)
			}
		})
	}
}

// TestUserAgentSurvivesTransportReplacement checks that a caller who swaps in
// their own transport still sends the User-Agent, because it is set on each
// request rather than only on the transport.
func TestUserAgentSurvivesTransportReplacement(t *testing.T) {
	server, seen := newUAServer(t)
	const callerUA = "App/1.0 (mailto:app@example.org)"
	client := newClientForServer(t, callerUA, server)
	client.Client = &http.Client{Transport: server.Client().Transport}

	if _, err := client.GetEntities(context.Background(), []string{"Q1"}, nil); err != nil {
		t.Fatalf("GetEntities: %s", err)
	}

	got := seen()
	if len(got) != 1 {
		t.Fatalf("server saw %d requests, want 1", len(got))
	}
	want := callerUA + " " + quickiedata.LibraryUserAgent
	if got[0] != want {
		t.Errorf("user agent = %q, want %q", got[0], want)
	}
}

// TestNewClientRequiresUserAgent checks that an unidentified client is
// rejected before it can send anything, and that the error says what to do.
func TestNewClientRequiresUserAgent(t *testing.T) {
	for _, userAgent := range []string{"", "   "} {
		client, err := quickiedata.NewClient(userAgent, nil)
		if err == nil {
			t.Fatalf("NewClient(%q) = client, want an error", userAgent)
		}
		if client != nil {
			t.Errorf("NewClient(%q) returned a client beside the error", userAgent)
		}
		if !strings.Contains(err.Error(), "user agent") {
			t.Errorf("error = %q, want it to name the user agent", err)
		}
		if !strings.Contains(err.Error(), "Policy:Wikimedia_Foundation_User-Agent_Policy") {
			t.Errorf("error = %q, want it to cite the policy", err)
		}
	}

	_, err := quickiedata.NewClient("App/1.0\r\nX-Injected: 1", nil)
	if err == nil {
		t.Fatal("expected an error for a user agent with a line break")
	}
	if !strings.Contains(err.Error(), "invalid characters") {
		t.Errorf("error = %q, want it to name the invalid characters", err)
	}
}

// TestDefaultTransportRetriesOn429 checks that the transport built by default,
// not a fake, retries a 429 and succeeds on the next attempt.
func TestDefaultTransportRetriesOn429(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the default backoff")
	}

	var mu sync.Mutex
	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		attempt := requests
		mu.Unlock()

		if attempt == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"entities":{"Q1":{"id":"Q1","type":"item"}},"success":1}`)
	}))
	t.Cleanup(server.Close)

	client, err := quickiedata.NewClient("quickiedata-test/0.0", nil)
	if err != nil {
		t.Fatalf("NewClient: %s", err)
	}
	client.APIEndpoint = server.URL

	result, err := client.GetEntities(context.Background(), []string{"Q1"}, nil)
	if err != nil {
		t.Fatalf("GetEntities: %s", err)
	}
	if _, ok := result.Entities["Q1"]; !ok {
		t.Error("entity Q1 missing from the parsed response")
	}

	mu.Lock()
	defer mu.Unlock()
	if requests != 2 {
		t.Errorf("server saw %d requests, want 2", requests)
	}
}
