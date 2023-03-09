package quickiedata

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type WikidataService struct {
	APIEndpoint    string
	SPARQLEndpoint string
	Client         *http.Client
}

func NewWikidataService() *WikidataService {
	// TODO: initialize http client with user agent
	// TODO: rate limiting and backoff
	return &WikidataService{
		APIEndpoint:    "https://www.wikidata.org/w/api.php",
		SPARQLEndpoint: "https://query.wikidata.org/sparql",
		Client:         http.DefaultClient,
	}
}

func (wd *WikidataService) GetEntitiesAsSimple(ids []string, opt *GetEntitiesOptions) (map[string]interface{}, error) {
	entities, err := wd.GetEntities(ids, opt)
	if err != nil {
		return nil, err
	}

	return SimplifyEntities(entities), nil
}

func (wd *WikidataService) GetEntities(ids []string, opt *GetEntitiesOptions) (map[string]*EntityInfo, error) {
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

	return result.Entities, nil
}

func (wd *WikidataService) GetEntitiesRaw(ids []string, opt *GetEntitiesOptions) ([]byte, error) {
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

func (wd *WikidataService) SearchEntities(query string, options *SearchEntitiesOptions) ([]*SearchResult, error) {
	rawBody, err := wd.SearchEntitiesRaw(query, options)
	if err != nil {
		return nil, err
	}

	var result SearchResponse
	err = json.Unmarshal(rawBody, &result)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return result.Search, nil
}

func (wd *WikidataService) SearchEntitiesRaw(query string, options *SearchEntitiesOptions) ([]byte, error) {
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

func (wd *WikidataService) GetSPARQLQueryAsSimple(query string, options *GetSPARQLQueryOptions) ([]map[string]interface{}, error) {
	entities, err := wd.GetSPARQLQuery(query, options)
	if err != nil {
		return nil, err
	}

	return SimplifySPARQLResults(entities), nil
}

func (wd *WikidataService) GetSPARQLQuery(query string, options *GetSPARQLQueryOptions) (*SPARQLResults, error) {
	rawBody, err := wd.GetSPARQLQueryRaw(query, options)
	if err != nil {
		return nil, err
	}

	var result SPARQLResults
	err = json.Unmarshal(rawBody, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (wd *WikidataService) GetSPARQLQueryRaw(query string, options *GetSPARQLQueryOptions) ([]byte, error) {
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

func (wd *WikidataService) CreateSPARQLQuery(sparqlQuery string, options *GetSPARQLQueryOptions) (string, error) {
	if len(sparqlQuery) == 0 {
		return "", errors.New("sparql query is empty")
	}

	sparqlQuery, err := RenderSPARQLQuery(sparqlQuery, options)
	if err != nil {
		return "", err
	}

	fmt.Println(sparqlQuery)
	query := url.Values{}
	query.Add("format", "json")
	query.Add("query", sparqlQuery)
	if options.Timeout > 0 {
		query.Add("timeout", strconv.FormatInt(options.Timeout, 10))
	}

	fullURL := wd.SPARQLEndpoint + "?" + query.Encode()
	return fullURL, nil
}

func (wd *WikidataService) CreateGetEntitiesURL(ids []string, opt *GetEntitiesOptions) (string, error) {
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

func (wd *WikidataService) CreateSearchEntitiesURL(search string, opt *SearchEntitiesOptions) (string, error) {
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
