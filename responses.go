package quickiedata

import (
	"encoding/json"
	"fmt"
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
	Entities map[string]interface{} `json:"entities,omitempty"`
}

func (resp *GetEntitiesSimpleResponse) UnmarshalJSON(data []byte) error {
	var peek struct {
		Entities map[string]json.RawMessage `json:"entities"`
	}
	err := json.Unmarshal(data, &peek)
	if err != nil {
		return err
	}

	for key, data := range peek.Entities {
		value, err := unmarshalSimpleEntity(data)
		if err != nil {
			return err
		}
		resp.Entities[key] = value
	}
	return nil
}

type GetEntityResponse struct {
	Entity *EntityInfo `json:"entity,omitempty"`
}

type GetEntitySimpleResponse struct {
	Entity interface{} `json:"entity,omitempty"`
}

func (resp *GetEntitiesResponse) Simplify() *GetEntitiesSimpleResponse {
	var output = make(map[string]interface{})
	for key, entity := range resp.Entities {
		simple := SimplifyEntity(entity)
		if simple != nil {
			output[key] = simple
		}
	}
	return &GetEntitiesSimpleResponse{
		Entities: output,
	}
}

func (resp *GetEntityResponse) Simplify() *GetEntitySimpleResponse {
	simple := SimplifyEntity(resp.Entity)
	if simple != nil {
		return &GetEntitySimpleResponse{
			Entity: simple,
		}
	}
	return nil
}

func (resp *GetEntitySimpleResponse) UnmarshalJSON(data []byte) error {
	var peek struct {
		Entity json.RawMessage `json:"entity"`
	}
	err := json.Unmarshal(data, &peek)
	if err != nil {
		return err
	}

	value, err := unmarshalSimpleEntity(peek.Entity)
	if err != nil {
		return err
	}
	resp.Entity = value
	return nil
}

func unmarshalSimpleEntity(data json.RawMessage) (interface{}, error) {
	var peek struct {
		Type string `json:"type"`
	}
	err := json.Unmarshal(data, &peek)
	if err != nil {
		return nil, err
	}

	switch peek.Type {
	case "item":
		var value SimpleItem
		err := json.Unmarshal(data, &value)
		if err != nil {
			return nil, err
		}
		return &value, nil
	case "property":
		var value SimpleProperty
		err := json.Unmarshal(data, &value)
		if err != nil {
			return nil, err
		}
		return &value, nil
	case "lexeme":
		var value SimpleLexeme
		err := json.Unmarshal(data, &value)
		if err != nil {
			return nil, err
		}
		return &value, nil
	case "form":
		var value SimpleForm
		err := json.Unmarshal(data, &value)
		if err != nil {
			return nil, err
		}
		return &value, nil
	case "sense":
		var value SimpleSense
		err := json.Unmarshal(data, &value)
		if err != nil {
			return nil, err
		}
		return &value, nil
	default:
		return nil, fmt.Errorf("%s entity value parser not implemented", peek.Type)
	}
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
