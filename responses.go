package quickiedata

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

type GetEntityResponse struct {
	Entity *EntityInfo
}

type GetEntitySimpleResponse struct {
	Entity interface{}
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

func (resp *GetEntitiesSimpleResponse) GetEntityAsItem(key string) *SimpleItem {
	if resp == nil {
		return nil
	}
	value, exists := resp.Entities[key]
	if !exists {
		return nil
	}
	casted, ok := value.(*SimpleItem)
	if !ok {
		return nil
	}
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
	casted, ok := value.(*SimpleProperty)
	if !ok {
		return nil
	}
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
	casted, ok := value.(*SimpleLexeme)
	if !ok {
		return nil
	}
	return casted
}

func (resp *GetEntitySimpleResponse) GetEntityAsItem(key string) *SimpleItem {
	if resp == nil {
		return nil
	}
	casted, ok := resp.Entity.(*SimpleItem)
	if !ok {
		return nil
	}
	return casted
}

func (resp *GetEntitySimpleResponse) GetEntityAsProperty(key string) *SimpleProperty {
	if resp == nil {
		return nil
	}
	casted, ok := resp.Entity.(*SimpleProperty)
	if !ok {
		return nil
	}
	return casted
}

func (resp *GetEntitySimpleResponse) GetEntityAsLexeme(key string) *SimpleLexeme {
	if resp == nil {
		return nil
	}
	casted, ok := resp.Entity.(*SimpleLexeme)
	if !ok {
		return nil
	}
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
