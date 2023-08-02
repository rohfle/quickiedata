package quickiedata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type WikidataClient struct {
	APIEndpoint    string
	SPARQLEndpoint string
	Client         *http.Client
}

func NewWikidataClient(settings *HTTPClientSettings) *WikidataClient {
	return &WikidataClient{
		APIEndpoint:    "https://www.wikidata.org/w/api.php",
		SPARQLEndpoint: "https://query.wikidata.org/sparql",
		Client:         QuickieHTTPClient(settings),
	}
}

func (wd *WikidataClient) GetWithContext(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

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
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
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
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
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

	req, err := http.NewRequestWithContext(ctx, "POST", wd.SPARQLEndpoint, strings.NewReader(sparqlQuery))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/sparql-query")
	req.Header.Set("Accept", "application/sparql-results+json")

	resp, err := wd.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request returned status: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Create a wikidata api get entries (wbgetentries) query url
func (wd *WikidataClient) CreateGetEntitiesURL(ids []string, opt *GetEntitiesOptions) (string, error) {
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

// Create a wikidata api search entries (wbsearchentries) query url
func (wd *WikidataClient) CreateSearchEntitiesURL(search string, opt *SearchEntitiesOptions) (string, error) {
	if len(search) == 0 {
		return "", errors.New("no ids specified")
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
