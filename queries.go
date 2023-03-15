package quickiedata

import (
	"encoding/json"
	"errors"
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

func (wd *WikidataClient) GetEntitiesRaw(ids []string, opt *GetEntitiesOptions) ([]byte, error) {
	url, err := wd.CreateGetEntitiesURL(ids, opt)
	if err != nil {
		return nil, err
	}

	resp, err := wd.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (wd *WikidataClient) GetEntities(ids []string, opt *GetEntitiesOptions) (*GetEntitiesResponse, error) {
	rawBody, err := wd.GetEntitiesRaw(ids, opt)
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

func (wd *WikidataClient) GetEntitiesSimple(ids []string, opt *GetEntitiesOptions) (*GetEntitiesSimpleResponse, error) {
	response, err := wd.GetEntities(ids, opt)
	if err != nil {
		return nil, err
	}

	return response.Simplify(), nil
}

func (wd *WikidataClient) GetEntity(id string, opt *GetEntitiesOptions) (*GetEntityResponse, error) {
	response, err := wd.GetEntities([]string{id}, opt)
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

func (wd *WikidataClient) GetEntitySimple(id string, opt *GetEntitiesOptions) (*GetEntitySimpleResponse, error) {
	response, err := wd.GetEntity(id, opt)
	if err != nil {
		return nil, err
	}

	return response.Simplify(), nil
}

func (wd *WikidataClient) SearchEntitiesRaw(query string, options *SearchEntitiesOptions) ([]byte, error) {
	url, err := wd.CreateSearchEntitiesURL(query, options)
	if err != nil {
		return nil, err
	}

	resp, err := wd.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (wd *WikidataClient) SearchEntities(query string, options *SearchEntitiesOptions) ([]*SearchResult, error) {
	rawBody, err := wd.SearchEntitiesRaw(query, options)
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

func (wd *WikidataClient) SPARQLQuerySimple(query *SPARQLQuery, options *GetSPARQLQueryOptions) (*SPARQLSimpleResponse, error) {
	response, err := wd.SPARQLQuery(query, options)
	if err != nil {
		return nil, err
	}

	return response.Simplify(), nil
}

func (wd *WikidataClient) SPARQLQuery(query *SPARQLQuery, options *GetSPARQLQueryOptions) (*SPARQLResponse, error) {
	rawBody, err := wd.SPARQLQueryRaw(query, options)
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

func (wd *WikidataClient) SPARQLQueryRaw(query *SPARQLQuery, options *GetSPARQLQueryOptions) ([]byte, error) {
	url, err := wd.CreateSPARQLQuery(query, options)
	if err != nil {
		return nil, err
	}

	resp, err := wd.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Create a sparql query and render it as a URL
func (wd *WikidataClient) CreateSPARQLQuery(query *SPARQLQuery, options *GetSPARQLQueryOptions) (string, error) {
	if query == nil || len(query.Template) == 0 {
		return "", errors.New("sparql query is empty")
	}

	sparqlQuery, err := RenderSPARQLQuery(query)
	if err != nil {
		return "", err
	}

	DebugLog.Printf("SPARQL query: %s\n", sparqlQuery)
	queryParams := url.Values{}
	queryParams.Add("format", "json")
	queryParams.Add("query", sparqlQuery)
	if options.Timeout > 0 {
		queryParams.Add("timeout", strconv.FormatInt(options.Timeout, 10))
	}

	fullURL := wd.SPARQLEndpoint + "?" + queryParams.Encode()
	return fullURL, nil
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
