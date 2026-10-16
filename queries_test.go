package quickiedata_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/rohfle/quickiedata"
)

// newTestClient starts a test server and returns a client that sends its API
// and SPARQL requests to that server. The test server stops at the end of the
// test.
func newTestClient(t *testing.T, handler http.HandlerFunc) *quickiedata.WikidataClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := quickiedata.NewClient(server.Client().Transport)
	client.APIEndpoint = server.URL + "/w/api.php"
	client.SPARQLEndpoint = server.URL + "/sparql"
	return client
}

func TestRawRequestsReportHTTPFailures(t *testing.T) {
	const reason = "Please set a user-agent and respect our robot policy"

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "<html>\n  <body>%s</body>\n</html>\n", reason)
	})

	query := quickiedata.NewSPARQLQuery()
	query.Template = "SELECT ?item WHERE { ?item wdt:P31 wd:Q146 }"

	tests := []struct {
		name string
		call func() ([]byte, error)
	}{
		{"GetEntitiesRaw", func() ([]byte, error) {
			return client.GetEntitiesRaw(context.Background(), []string{"Q1"}, quickiedata.NewGetEntitiesOptions())
		}},
		{"SearchEntitiesRaw", func() ([]byte, error) {
			return client.SearchEntitiesRaw(context.Background(), "hubble", quickiedata.NewSearchEntitiesOptions())
		}},
		{"SPARQLQueryRaw", func() ([]byte, error) {
			return client.SPARQLQueryRaw(context.Background(), query, quickiedata.NewSPARQLQueryOptions())
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := test.call()
			if err == nil {
				t.Fatalf("expected an error for a 403 response, got body %q", body)
			}
			if body != nil {
				t.Errorf("expected no body alongside the error, got %q", body)
			}
			for _, want := range []string{"403", "Forbidden", reason} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "\n") {
				t.Errorf("error should be a single line, got %q", err)
			}
		})
	}
}

func TestStatusErrorReachesCallers(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":{"code":"ratelimited","info":"too many requests"}}`)
	})

	_, err := client.GetEntities(context.Background(), []string{"Q1"}, quickiedata.NewGetEntitiesOptions())
	if err == nil {
		t.Fatal("expected GetEntities to report the 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("status missing from error: %q", err)
	}
}

func TestGetEntitiesSuccessStillParses(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"entities":{"Q1":{"id":"Q1","type":"item","labels":{"en":{"language":"en","value":"Universe"}}}},"success":1}`)
	})

	response, err := client.GetEntities(context.Background(), []string{"Q1"}, quickiedata.NewGetEntitiesOptions())
	if err != nil {
		t.Fatalf("GetEntities: %s", err)
	}
	entity, ok := response.Entities["Q1"]
	if !ok {
		t.Fatalf("Q1 missing from %+v", response.Entities)
	}
	if got := entity.Labels["en"].Value; got != "Universe" {
		t.Errorf("label = %q, want %q", got, "Universe")
	}

	simple, err := json.Marshal(response.Simplify())
	if err != nil {
		t.Fatalf("marshal simplified response: %s", err)
	}
	if !strings.Contains(string(simple), `"Universe"`) {
		t.Errorf("simplified output lost the label: %s", simple)
	}
}

func TestGetEntitySimpleMissingEntity(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"entities":{"Q999999999":{"id":"Q999999999","missing":""}},"success":1}`)
	})

	response, err := client.GetEntitySimple(context.Background(), "Q999999999", quickiedata.NewGetEntitiesOptions())
	if err != nil {
		t.Fatalf("GetEntitySimple: %s", err)
	}
	if response != nil {
		t.Errorf("expected nil response for a missing entity, got %+v", response)
	}
	// Make sure that a nil response also returns a nil item. This keeps the
	// call sequence in the README safe.
	if item := response.GetEntityAsItem("Q999999999"); item != nil {
		t.Errorf("expected nil item, got %+v", item)
	}
}

// An entity that this library cannot simplify must be reported, not dropped.
func TestGetEntitiesSimpleReportsDroppedEntities(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"entities":{
			"Q42":{"id":"Q42","type":"item","labels":{"en":{"language":"en","value":"Douglas Adams"}}},
			"Q1":null,
			"Q999999998":{"id":"Q999999998","missing":""},
			"Q999999999":{"id":"Q999999999","missing":""},
			"M123":{"id":"M123","type":"mediainfo"}
		},"success":1}`)
	})

	ids := []string{"Q42", "Q1", "Q999999998", "Q999999999", "M123"}
	response, err := client.GetEntitiesSimple(context.Background(), ids, nil)
	if err != nil {
		t.Fatalf("GetEntitiesSimple: %s", err)
	}

	if item := response.GetEntityAsItem("Q42"); item == nil || item.Labels["en"] != "Douglas Adams" {
		t.Errorf("Q42 was not simplified: %+v", response.Entities["Q42"])
	}
	if want := []string{"Q1", "Q999999998", "Q999999999"}; !slices.Equal(response.Missing, want) {
		t.Errorf("Missing = %v, want %v", response.Missing, want)
	}
	if want := map[string]string{"M123": "mediainfo"}; !maps.Equal(response.Unsupported, want) {
		t.Errorf("Unsupported = %v, want %v", response.Unsupported, want)
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}
	for _, want := range []string{`"missing":["Q1","Q999999998","Q999999999"]`, `"unsupported":{"M123":"mediainfo"}`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("json %s does not contain %s", data, want)
		}
	}
}

func TestGetEntitySimpleUnsupportedType(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"entities":{"M123":{"id":"M123","type":"mediainfo"}},"success":1}`)
	})

	response, err := client.GetEntitySimple(context.Background(), "M123", nil)
	if err != nil {
		t.Fatalf("GetEntitySimple: %s", err)
	}
	if response == nil {
		t.Fatal("expected a response that names the unsupported type, got nil")
	}
	if response.Unsupported != "mediainfo" {
		t.Errorf("Unsupported = %q, want %q", response.Unsupported, "mediainfo")
	}
	if response.Entity != nil {
		t.Errorf("Entity should stay nil, got %+v", response.Entity)
	}
	if item := response.GetEntityAsItem("M123"); item != nil {
		t.Errorf("expected a nil item, got %+v", item)
	}
}

func TestNilOptionsAreDefaulted(t *testing.T) {
	client := quickiedata.NewClient(nil)

	got, err := client.CreateGetEntitiesURL([]string{"Q42"}, nil)
	if err != nil {
		t.Fatalf("CreateGetEntitiesURL: %s", err)
	}
	if !strings.Contains(got, "format=json") {
		t.Errorf("url = %s", got)
	}

	got, err = client.CreateSearchEntitiesURL("hubble", nil)
	if err != nil {
		t.Fatalf("CreateSearchEntitiesURL: %s", err)
	}
	if !strings.Contains(got, "type=item") {
		t.Errorf("url = %s", got)
	}

	server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"entities":{"Q42":{"id":"Q42","type":"item"}},"search":[{"id":"Q42","label":"Douglas Adams"}],"success":1}`)
	})
	if _, err := server.GetEntities(context.Background(), []string{"Q42"}, nil); err != nil {
		t.Errorf("GetEntities with nil options: %s", err)
	}
	if _, err := server.SearchEntities(context.Background(), "hubble", nil); err != nil {
		t.Errorf("SearchEntities with nil options: %s", err)
	}
}
