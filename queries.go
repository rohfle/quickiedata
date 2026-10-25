package quickiedata

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rohfle/nicehttp"
)

// Wikidata Query Service endpoints. The service is migrating from v1 to v2,
// which changes the endpoint URL and the supported SPARQL; see
// SPARQLCompatibilityIssues and the README section "Migrating to WDQS v2".
const (
	// SPARQLEndpointV1 is the current WDQS endpoint. It stays available but
	// may degrade from February 2027.
	SPARQLEndpointV1 = "https://query.wikidata.org/sparql"
	// SPARQLEndpointV2 is the new WDQS endpoint.
	SPARQLEndpointV2 = "https://query-next.wikidata.org/sparql"
)

// libraryName prefixes the library's own user agent token.
const libraryName = "quickiedata"

// userAgentPolicyURL is where callers are pointed when the user agent is
// missing, since an unidentified client gets a 403 from Wikimedia.
const userAgentPolicyURL = "https://foundation.wikimedia.org/wiki/Policy:Wikimedia_Foundation_User-Agent_Policy"

var LibraryUserAgent = fmt.Sprintf("%s/%s (https://github.com/rohfle/quickiedata)", libraryName, getVersion())

type WikidataClient struct {
	APIEndpoint    string
	SPARQLEndpoint string
	Client         *http.Client

	// userAgent is set by NewClient and added to every request. A client built
	// by hand instead of through NewClient sends no User-Agent of its own.
	userAgent string
}

type WikidataClientOptions struct {
	APIEndpoint         string
	SPARQLEndpoint      string
	Backoff             *nicehttp.Backoff
	MaxAttempts         int
	AttemptTimeout      time.Duration
	DownstreamTransport http.RoundTripper
	SuppressLibraryUA   bool
}

// isValidUserAgent reports whether s is a legal HTTP header value.
// Per RFC 7230 §3.2.6, field values may contain visible ASCII (0x21–0x7E),
// spaces, and horizontal tabs; they must NOT contain CR, LF, or NUL.
func isValidUserAgent(s string) bool {
	for _, r := range s {
		switch {
		case r == '\t' || r == ' ':
			// allowed
		case r >= 0x21 && r <= 0x7E:
			// visible ASCII
		default:
			return false
		}
	}
	return true
}

// buildUserAgent validates the caller-supplied user agent and appends the
// library's own identifier per Wikimedia's User-Agent policy. The result is
// "<caller> <library>" — tool first, library second, matching the convention
// documented at https://meta.wikimedia.org/wiki/User-Agent_policy.
//
// If suppressLibraryUA is set, the library identifier is NOT appended and
// the caller is solely responsible for including it. This is a policy
// violation unless the caller has a good reason (proxying, exact-header tests).
func buildUserAgent(callerUA string, suppressLibraryUA bool) (string, error) {
	callerUA = strings.TrimSpace(callerUA)
	if callerUA == "" {
		return "", fmt.Errorf("user agent must not be empty, identify your application as %q: %s",
			"app/1.0 (you@example.org)", userAgentPolicyURL)
	}
	if !isValidUserAgent(callerUA) {
		return "", fmt.Errorf("user agent contains invalid characters: %q", callerUA)
	}

	if suppressLibraryUA {
		return callerUA, nil
	}

	// Caller may have already included our identifier (e.g. a wrapper library
	// that concatenates, or a caller pinning the version itself). Trust them
	// rather than emit a duplicate token. The match is on "quickiedata/", so
	// a caller named "quickiedata-cli/0.1" still gets the token.
	if strings.Contains(callerUA, libraryName+"/") {
		return callerUA, nil
	}

	return callerUA + " " + LibraryUserAgent, nil
}

func NewClient(userAgent string, options *WikidataClientOptions) (*WikidataClient, error) {
	if options == nil {
		options = &WikidataClientOptions{}
	}

	userAgent, err := buildUserAgent(userAgent, options.SuppressLibraryUA)
	if err != nil {
		return nil, err
	}

	transport := nicehttp.NewNiceTransportBuilder().
		SetUserAgent(userAgent).
		SetLimiterBackoff(cmp.Or(options.Backoff, nicehttp.NewBackoff(1*time.Second, 120*time.Second, nicehttp.DefaultBackoffCoefficients))).
		SetMaxAttempts(cmp.Or(options.MaxAttempts, 5)).
		SetAttemptTimeout(cmp.Or(options.AttemptTimeout, 60*time.Second)).
		SetDownstreamTransport(options.DownstreamTransport).
		Build()

	return &WikidataClient{
		APIEndpoint:    cmp.Or(options.APIEndpoint, "https://www.wikidata.org/w/api.php"),
		SPARQLEndpoint: cmp.Or(options.SPARQLEndpoint, SPARQLEndpointV1),
		Client:         &http.Client{Transport: transport},
		userAgent:      userAgent,
	}, nil
}

// setUserAgent labels a request with the client's User-Agent. Setting it on the
// request, not only on the transport, keeps the label when a caller replaces
// Client.Transport with one of their own; a header the caller already set is
// left alone.
func (wd *WikidataClient) setUserAgent(req *http.Request) {
	if wd.userAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", wd.userAgent)
	}
}

// errorBodyLimit is the maximum number of bytes that readResponseBody reads
// from the body of a failed response. The error message is made from this
// part of the body.
const errorBodyLimit = 512

// readResponseBody reads the body of a response and then closes the body.
// If the status code is 300 or more, the function returns an error and no
// body. The error message contains the status code and part of the body.
// If the body is empty, the error message contains only the status code.
// The function replaces each group of space characters with one space.
// If the read stops with an error, the function returns the part of the
// body that it read and that error.
func readResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		message := strings.Join(strings.Fields(string(snippet)), " ")
		if message == "" {
			return nil, fmt.Errorf("request returned status %s", resp.Status)
		}
		return nil, fmt.Errorf("request returned status %s: %s", resp.Status, message)
	}

	return io.ReadAll(resp.Body)
}

// GetWithContext sends a GET request and returns the response to the caller.
// This function does not check the status code. It does not read the body.
// The caller must check the status code, read the body and close the body.
// GetEntitiesRaw, SearchEntitiesRaw and SPARQLQueryRaw do this work.
// The client's User-Agent is added to the request.
func (wd *WikidataClient) GetWithContext(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	wd.setUserAgent(req)

	return wd.Client.Do(req)
}

func (wd *WikidataClient) GetEntitiesRaw(ctx context.Context, ids []string, opt *GetEntitiesOptions) ([]byte, error) {
	url, err := wd.CreateGetEntitiesURL(ids, opt)
	if err != nil {
		return nil, err
	}

	resp, err := wd.GetWithContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return readResponseBody(resp)
}

func (wd *WikidataClient) GetEntities(ctx context.Context, ids []string, options *GetEntitiesOptions) (*GetEntitiesResponse, error) {
	rawBody, err := wd.GetEntitiesRaw(ctx, ids, options)
	if err != nil {
		return nil, err
	}

	var result GetEntitiesResponse
	err = json.Unmarshal(rawBody, &result)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &result, nil
}

func (wd *WikidataClient) GetEntitiesSimple(ctx context.Context, ids []string, options *GetEntitiesOptions) (*GetEntitiesSimpleResponse, error) {
	response, err := wd.GetEntities(ctx, ids, options)
	if err != nil {
		return nil, err
	}

	return response.Simplify(), nil
}

func (wd *WikidataClient) GetEntity(ctx context.Context, id string, options *GetEntitiesOptions) (*GetEntityResponse, error) {
	response, err := wd.GetEntities(ctx, []string{id}, options)
	if err != nil {
		return nil, err
	}

	entity, ok := response.Entities[id]
	if !ok {
		return nil, nil // not found
	}

	return &GetEntityResponse{
		Entity: entity,
	}, nil
}

// GetEntitySimple returns the simplified entity. The function returns a nil
// response and a nil error when the API does not return the entity. It also
// returns a nil response and a nil error when this library does not support
// the entity type.
func (wd *WikidataClient) GetEntitySimple(ctx context.Context, id string, options *GetEntitiesOptions) (*GetEntitySimpleResponse, error) {
	response, err := wd.GetEntity(ctx, id, options)
	if err != nil {
		return nil, err
	}

	return response.Simplify(), nil
}

func (wd *WikidataClient) SearchEntitiesRaw(ctx context.Context, query string, options *SearchEntitiesOptions) ([]byte, error) {
	url, err := wd.CreateSearchEntitiesURL(query, options)
	if err != nil {
		return nil, err
	}

	resp, err := wd.GetWithContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return readResponseBody(resp)
}

func (wd *WikidataClient) SearchEntities(ctx context.Context, query string, options *SearchEntitiesOptions) ([]*SearchResult, error) {
	rawBody, err := wd.SearchEntitiesRaw(ctx, query, options)
	if err != nil {
		return nil, err
	}

	var result SearchEntitiesResponse
	err = json.Unmarshal(rawBody, &result)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return result.Search, nil
}

func (wd *WikidataClient) SPARQLQuerySimple(ctx context.Context, query *SPARQLQuery, options *GetSPARQLQueryOptions) (*SPARQLSimpleResponse, error) {
	response, err := wd.SPARQLQuery(ctx, query, options)
	if err != nil {
		return nil, err
	}

	return response.Simplify(), nil
}

func (wd *WikidataClient) SPARQLQuery(ctx context.Context, query *SPARQLQuery, options *GetSPARQLQueryOptions) (*SPARQLResponse, error) {
	rawBody, err := wd.SPARQLQueryRaw(ctx, query, options)
	if err != nil {
		return nil, err
	}

	var result SPARQLResponse
	err = json.Unmarshal(rawBody, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (wd *WikidataClient) SPARQLQueryRaw(ctx context.Context, query *SPARQLQuery, options *GetSPARQLQueryOptions) ([]byte, error) {
	sparqlQuery, err := RenderSPARQLQuery(query)
	if err != nil {
		return nil, err
	}

	// The query is sent whatever it contains: the endpoint in use decides what
	// it accepts, and WDQS v1 still accepts these constructs.
	for _, issue := range SPARQLCompatibilityIssues(sparqlQuery) {
		DebugLog.Printf("sparql: %s is not supported by WDQS v2: %s (%s)",
			issue.Construct, issue.Message, issue.DocURL)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", wd.SPARQLEndpoint, strings.NewReader(sparqlQuery))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/sparql-query")
	req.Header.Set("Accept", "application/sparql-results+json")
	wd.setUserAgent(req)

	resp, err := wd.Client.Do(req)
	if err != nil {
		return nil, err
	}
	return readResponseBody(resp)
}

// CreateGetEntitiesURL creates a wikidata api get entries (wbgetentries) query url
func (wd *WikidataClient) CreateGetEntitiesURL(ids []string, opt *GetEntitiesOptions) (string, error) {
	if opt == nil {
		opt = NewGetEntitiesOptions()
	}
	if len(ids) == 0 {
		return "", errors.New("no ids specified")
	}
	if err := ValidateEntityIDs(ids); err != nil {
		return "", err
	}

	query := url.Values{}
	query.Add("action", "wbgetentities")
	query.Add("ids", strings.Join(ids, "|"))
	if opt.Format == "" {
		query.Add("format", "json")
	} else {
		query.Add("format", opt.Format)
	}
	if len(opt.Props) > 0 {
		query.Add("props", strings.Join(opt.Props, "|"))
	}
	if len(opt.Languages) > 0 {
		query.Add("languages", strings.Join(opt.Languages, "|"))
	}
	if len(opt.Sitefilter) > 0 {
		query.Add("sitefilter", strings.Join(opt.Sitefilter, "|"))
	}
	if !opt.Redirects {
		query.Add("redirects", "no")
	}

	fullURL := wd.APIEndpoint + "?" + query.Encode()
	return fullURL, nil
}

// CreateSearchEntitiesURL creates a wikidata api search entries (wbsearchentries) query url
func (wd *WikidataClient) CreateSearchEntitiesURL(search string, opt *SearchEntitiesOptions) (string, error) {
	if opt == nil {
		opt = NewSearchEntitiesOptions()
	}
	if len(search) == 0 {
		return "", errors.New("no search term specified")
	}
	if err := ValidateEntityType(opt.EntityType); err != nil {
		return "", err
	}

	query := url.Values{}
	query.Add("action", "wbsearchentities")
	query.Add("search", search)
	query.Add("type", opt.EntityType)
	query.Add("limit", strconv.FormatInt(opt.Limit, 10))
	query.Add("continue", strconv.FormatInt(opt.Offset, 10))

	query.Add("language", opt.Language)

	if opt.UseLang != "" {
		query.Add("uselang", opt.UseLang)
	} else {
		query.Add("uselang", opt.Language)
	}

	if opt.Format == "" {
		query.Add("format", "json")
	} else {
		query.Add("format", opt.Format)
	}

	fullURL := wd.APIEndpoint + "?" + query.Encode()
	return fullURL, nil

}
