package quickiedata

import (
	"encoding/json"
	"fmt"
	"slices"
)

type ResponseError struct {
	Code  string `json:"code"`
	Info  string `json:"info"`
	Extra string `json:"extra"`
}

func (re *ResponseError) Error() string {
	return re.Info
}

type GetEntitiesResponse struct {
	Entities map[string]*EntityInfo `json:"entities,omitempty"`
	Success  int64                  `json:"success"`
	Error    *ResponseError         `json:"error,omitempty"`
}

type GetEntitiesSimpleResponse struct {
	Entities map[string]any `json:"entities,omitempty"`
	// Missing lists the requested ids that do not exist.
	Missing []string `json:"missing,omitempty"`
	// Unsupported maps an entity id to the type that this library cannot
	// simplify, for example "mediainfo".
	Unsupported map[string]string `json:"unsupported,omitempty"`
}

func (resp *GetEntitiesSimpleResponse) UnmarshalJSON(data []byte) error {
	var peek struct {
		Entities    map[string]json.RawMessage `json:"entities"`
		Missing     []string                   `json:"missing"`
		Unsupported map[string]string          `json:"unsupported"`
	}
	err := json.Unmarshal(data, &peek)
	if err != nil {
		return err
	}

	resp.Entities = make(map[string]any, len(peek.Entities))
	for key, data := range peek.Entities {
		value, err := unmarshalSimpleEntity(data)
		if err != nil {
			return err
		}
		resp.Entities[key] = value
	}
	resp.Missing = peek.Missing
	resp.Unsupported = peek.Unsupported
	return nil
}

type GetEntityResponse struct {
	Entity *EntityInfo `json:"entity,omitempty"`
}

type GetEntitySimpleResponse struct {
	Entity any `json:"entity,omitempty"`
	// Unsupported is the type of an entity that this library cannot
	// simplify, for example "mediainfo". The Entity field stays nil.
	Unsupported string `json:"unsupported,omitempty"`
}

func (resp *GetEntitiesResponse) Simplify() *GetEntitiesSimpleResponse {
	if resp == nil {
		return nil
	}

	var output = make(map[string]any)
	var missing []string
	var unsupported = make(map[string]string)

	for key, entity := range resp.Entities {
		simple := SimplifyEntity(entity)
		switch {
		case simple != nil:
			output[key] = simple
		case entity == nil, entity.Missing != nil:
			missing = append(missing, key)
		default:
			unsupported[key] = entity.Type
		}
	}
	slices.Sort(missing)

	return &GetEntitiesSimpleResponse{
		Entities:    output,
		Missing:     missing,
		Unsupported: unsupported,
	}
}

func (resp *GetEntityResponse) Simplify() *GetEntitySimpleResponse {
	if resp == nil || resp.Entity == nil || resp.Entity.Missing != nil {
		return nil
	}

	simple := SimplifyEntity(resp.Entity)
	if simple == nil {
		return &GetEntitySimpleResponse{
			Unsupported: resp.Entity.Type,
		}
	}
	return &GetEntitySimpleResponse{
		Entity: simple,
	}
}

func (resp *GetEntitySimpleResponse) UnmarshalJSON(data []byte) error {
	var peek struct {
		Entity      json.RawMessage `json:"entity"`
		Unsupported string          `json:"unsupported"`
	}
	err := json.Unmarshal(data, &peek)
	if err != nil {
		return err
	}

	resp.Unsupported = peek.Unsupported
	value, err := unmarshalSimpleEntity(peek.Entity)
	if err != nil {
		return err
	}
	resp.Entity = value
	return nil
}

func unmarshalSimpleEntity(data json.RawMessage) (any, error) {
	var peek struct {
		Type string `json:"type"`
	}
	err := json.Unmarshal(data, &peek)
	if err != nil {
		return nil, err
	}

	var value any

	switch peek.Type {
	case "item":
		value = &SimpleItem{}
	case "property":
		value = &SimpleProperty{}
	case "lexeme":
		value = &SimpleLexeme{}
	case "form":
		value = &SimpleForm{}
	case "sense":
		value = &SimpleSense{}
	default:
		return nil, fmt.Errorf("%s entity value parser not implemented", peek.Type)
	}

	err = json.Unmarshal(data, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (resp *GetEntitiesSimpleResponse) GetEntityAsItem(key string) *SimpleItem {
	if resp == nil {
		return nil
	}
	value, exists := resp.Entities[key]
	if !exists {
		return nil
	}
	casted, _ := value.(*SimpleItem)
	return casted
}

func (resp *GetEntitiesSimpleResponse) GetEntityAsProperty(key string) *SimpleProperty {
	if resp == nil {
		return nil
	}
	value, exists := resp.Entities[key]
	if !exists {
		return nil
	}
	casted, _ := value.(*SimpleProperty)
	return casted
}

func (resp *GetEntitiesSimpleResponse) GetEntityAsLexeme(key string) *SimpleLexeme {
	if resp == nil {
		return nil
	}
	value, exists := resp.Entities[key]
	if !exists {
		return nil
	}
	casted, _ := value.(*SimpleLexeme)
	return casted
}

func (resp *GetEntitySimpleResponse) GetEntityAsItem(key string) *SimpleItem {
	if resp == nil {
		return nil
	}
	casted, _ := resp.Entity.(*SimpleItem)
	return casted
}

func (resp *GetEntitySimpleResponse) GetEntityAsProperty(key string) *SimpleProperty {
	if resp == nil {
		return nil
	}
	casted, _ := resp.Entity.(*SimpleProperty)
	return casted
}

func (resp *GetEntitySimpleResponse) GetEntityAsLexeme(key string) *SimpleLexeme {
	if resp == nil {
		return nil
	}
	casted, _ := resp.Entity.(*SimpleLexeme)
	return casted
}

type SPARQLResponse struct {
	Head struct {
		Vars []string
	}
	Results struct {
		Bindings []map[string]*BindingValue
	}
}

func (results *SPARQLResponse) Simplify() *SPARQLSimpleResponse {
	var output []map[string]*SimpleBindingValue
	for _, binding := range results.Results.Bindings {
		var newResult = make(map[string]*SimpleBindingValue)
		for key, bvalue := range binding {
			if bvalue.Value == nil {
				continue
			}
			val, err := SimplifyBindingValue(bvalue)
			if err != nil {
				Log.Printf("error while simplifying %s value %v: %s\n", bvalue.DataType, *bvalue.Value, err)
				continue
			} else if val == nil {
				continue
			}
			newResult[key] = val
		}
		output = append(output, newResult)
	}
	return &SPARQLSimpleResponse{
		Results: output,
	}
}

type SPARQLSimpleResponse struct {
	Results []map[string]*SimpleBindingValue
}

type SearchEntitiesResponse struct {
	Search         []*SearchResult `json:"search"`
	SearchContinue int64           `json:"search-continue"`
	SearchInfo     struct {
		Search string `json:"search"`
	} `json:"searchinfo"`
	Success  int64          `json:"success"`
	Error    *ResponseError `json:"error"`
	ServedBy string         `json:"servedby"`
}

type SearchResult struct {
	Aliases     []string `json:"aliases,omitempty"`
	ConceptURI  string   `json:"concepturi"`
	Description string   `json:"description"`
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Match       struct {
		Language string `json:"language"`
		Text     string `json:"text"`
		Type     string `json:"type"`
	} `json:"match"`
	PageID     int64  `json:"pageid"`
	Repository string `json:"repository"`
	Title      string `json:"title"`
	URL        string `json:"url"`
}
