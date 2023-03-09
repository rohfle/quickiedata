package quickiedata

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ResultFormat string

const (
	FormatJSON ResultFormat = "json"
	FormatXML  ResultFormat = "xml"
)

type Rank string

const (
	RankNormal     Rank = "normal"
	RankPreferred  Rank = "preferred"
	RankDeprecated Rank = "deprecated"
)

type SnakType string

const (
	SnakTypeValue     SnakType = "value"
	SnakTypeSomeValue SnakType = "somevalue"
	SnakTypeNoValue   SnakType = "novalue"
)

// Wikidata datatype
// see https://www.wikidata.org/wiki/Special:ListDatatypes for more info
type DataType string

const (
	DataTypeCommonsMedia     DataType = "commonsMedia"
	DataTypeExternalID       DataType = "external-id"
	DataTypeGeoShape         DataType = "geo-shape"
	DataTypeGlobeCoordinate  DataType = "globecoordinate"
	DataTypeMath             DataType = "math"
	DataTypeMonolingualText  DataType = "monolingualtext"
	DataTypeMusicalNotation  DataType = "musical-notation"
	DataTypeQuantity         DataType = "quantity"
	DataTypeString           DataType = "string"
	DataTypeTabularData      DataType = "tabular-data"
	DataTypeTime             DataType = "time"
	DataTypeURL              DataType = "url"
	DataTypeWikibaseEntityID DataType = "wikibase-entityid"
	DataTypeWikibaseForm     DataType = "wikibase-form"
	DataTypeWikibaseItem     DataType = "wikibase-item"
	DataTypeWikibaseLexeme   DataType = "wikibase-lexeme"
	DataTypeWikibaseProperty DataType = "wikibase-property"
	DataTypeWikibaseSense    DataType = "wikibase-sense"
)

// Internal aliases that simplify matters, but aren't used by wikidata
const (
	DataTypeSimple DataType = "simple"
	DataTypeEntity DataType = "entity"
)

var DATATYPES_SIMPLE = []DataType{
	DataTypeCommonsMedia,
	DataTypeExternalID,
	DataTypeGeoShape,
	DataTypeMath,
	DataTypeMusicalNotation,
	DataTypeString,
	DataTypeTabularData,
	DataTypeURL,
}

var DATATYPES_ENTITY = []DataType{
	DataTypeWikibaseEntityID,
	DataTypeWikibaseForm,
	DataTypeWikibaseItem,
	DataTypeWikibaseLexeme,
	DataTypeWikibaseProperty,
	DataTypeWikibaseSense,
}

func DataTypeIsSimple(dtype DataType) bool {
	return ValueInSlice(dtype, DATATYPES_SIMPLE)
}
func DataTypeIsEntity(dtype DataType) bool {
	return ValueInSlice(dtype, DATATYPES_ENTITY)
}

func ValueInSlice[T comparable](needle T, haystack []T) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

type NumberPlus json.Number

func (n *NumberPlus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	str = strings.TrimPrefix(str, "+")
	*n = NumberPlus(str)
	return nil
}

type SPARQLResults struct {
	Head struct {
		Vars []string
	}
	Results struct {
		Bindings []map[string]*SPARQLBindingValue
	}
}

type SPARQLBindingValue struct {
	Value    *string `json:"value"`
	Type     string  `json:"type"`
	DataType string  `json:"datatype,omitempty"`
	Lang     string  `json:"xml:lang,omitempty"`
}

type SimpleSPARQLResults struct {
	Results []map[string]interface{}
}

type Snak struct {
	ID        string     `json:"id"`
	DataType  DataType   `json:"datatype"`
	DataValue *SnakValue `json:"datavalue"`
	Hash      string     `json:"hash"`
	Property  string     `json:"property"`
	SnakType  SnakType   `json:"snaktype"`
}

type SnakValueGlobeCoordinate struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Altitude  *float64 `json:"altitude,omitempty"`
	Precision float64  `json:"precision"`
	Globe     string   `json:"globe,omitempty"`
}

type SnakValueMonolingualText struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type SnakValueTime struct {
	After         int64  `json:"after,omitempty"`
	Before        int64  `json:"before,omitempty"`
	CalendarModel string `json:"calendarmodel,omitempty"`
	Precision     int    `json:"precision,omitempty"`
	Time          string `json:"time"`
	Timezone      int    `json:"timezone,omitempty"`
}

type SnakValueQuantity struct {
	Amount     NumberPlus `json:"amount"`
	Unit       string     `json:"unit,omitempty"`
	UpperBound NumberPlus `json:"upperbound,omitempty"`
	LowerBound NumberPlus `json:"lowerbound,omitempty"`
}

type SnakValueEntity struct {
	ID         string `json:"id"`
	NumericID  int64  `json:"numeric-id"`
	EntityType string `json:"entity-type"`
}

type SnakValue struct {
	Type  DataType    `json:"type"`
	Value interface{} `json:"value"`
}

func (sv *SnakValue) UnmarshalJSON(data []byte) error {
	var peek struct {
		Type  string
		Value json.RawMessage
	}
	err := json.Unmarshal(data, &peek)
	if err != nil {
		return err
	}

	// handle legacy "musical notation" datatype
	peek.Type = strings.ReplaceAll(peek.Type, " ", "-")
	sv.Type = DataType(peek.Type)

	workingType := sv.Type
	if DataTypeIsSimple(sv.Type) {
		workingType = DataTypeSimple
	} else if DataTypeIsEntity(sv.Type) {
		workingType = DataTypeEntity
	}

	switch workingType {
	case DataTypeSimple:
		var value string
		err := json.Unmarshal(peek.Value, &value)
		if err != nil {
			return err
		}
		sv.Value = value
	case DataTypeEntity:
		var value SnakValueEntity
		err := json.Unmarshal(peek.Value, &value)
		if err != nil {
			return err
		}
		sv.Value = &value
	case DataTypeGlobeCoordinate:
		var value SnakValueGlobeCoordinate
		err := json.Unmarshal(peek.Value, &value)
		if err != nil {
			return err
		}
		sv.Value = &value
	case DataTypeMonolingualText:
		var value SnakValueMonolingualText
		err := json.Unmarshal(peek.Value, &value)
		if err != nil {
			return err
		}
		sv.Value = &value
	case DataTypeQuantity:
		var value SnakValueQuantity
		err := json.Unmarshal(peek.Value, &value)
		if err != nil {
			return err
		}
		// remove leading + to prevent json.Number from failing to convert
		value.Amount = NumberPlus(strings.TrimPrefix(string(value.Amount), "+"))
		sv.Value = &value
	case DataTypeTime:
		var value SnakValueTime
		err := json.Unmarshal(peek.Value, &value)
		if err != nil {
			return err
		}
		sv.Value = &value
	default:
		return fmt.Errorf("%s snak value parser not implemented", workingType)
	}

	return nil
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

type SearchResponse struct {
	Search         []*SearchResult `json:"search"`
	SearchContinue int64           `json:"search-continue"`
	SearchInfo     struct {
		Search string `json:"search"`
	} `json:"searchinfo"`
	Success  int64          `json:"success"`
	Error    *ResponseError `json:"error"`
	ServedBy string         `json:"servedby"`
}

type GetEntitiesResponse struct {
	Entities map[string]*EntityInfo `json:"entities,omitempty"`
	Success  int64                  `json:"success"`
	Error    *ResponseError         `json:"error,omitempty"`
}

// This is a combined struct for all entity info types
// Use the Type field to work out which fields will have values
type EntityInfo struct {
	PageID    int64     `json:"pageid"`
	NS        int64     `json:"ns"`
	Title     string    `json:"title"`
	LastRevID int64     `json:"lastrevid"`
	Modified  string    `json:"modified"`
	Redirects *Redirect `json:"redirects,omitempty"`
	Type      string    `json:"type"`

	// Item fields
	ID           string               `json:"id"`
	Labels       map[string]*Term     `json:"labels"`
	Descriptions map[string]*Term     `json:"descriptions"`
	Aliases      map[string][]*Term   `json:"aliases"`
	Claims       map[string][]*Claim  `json:"claims"`
	Sitelinks    map[string]*Sitelink `json:"sitelinks"`

	// Property fields
	// ID (defined above)
	DataType DataType `json:"datatype"`
	// Labels (defined above)
	// Descriptions (defined above)
	// Aliases (defined above)
	// Claims (defined above)

	// Lexeme fields
	// ID (defined above)
	// DataType (defined above)
	LexicalCategory string           `json:"lexical-category"`
	Language        string           `json:"language"`
	Lemmas          map[string]*Term `json:"lemmas"`
	Forms           []*Form          `json:"forms"`
	Senses          []*Sense         `json:"senses"`

	// Form fields
	// ID (defined above)
	Representations     map[string]*Term `json:"representations"`
	GrammaticalFeatures []string         `json:"grammatical-features"`
	// Claims (defined above)

	// Sense fields
	// ID (defined above)
	Glosses map[string]*Term `json:"glosses"`
	// Claims (defined above)
}

type Claim struct {
	ID              string             `json:"id"`
	MainSnak        *Snak              `json:"mainsnak"`
	Rank            Rank               `json:"rank"`
	Type            DataType           `json:"type"`
	Qualifiers      map[string][]*Snak `json:"qualifiers"`
	QualifiersOrder []string           `json:"qualifiersOrder"`
	References      []*Reference       `json:"references"`
}

type Form struct {
	ID                  string              `json:"id"`
	Representations     map[string]*Term    `json:"representations"`
	GrammaticalFeatures []string            `json:"grammaticalFeatures"`
	Claims              map[string][]*Claim `json:"claims"`
}

type Sense struct {
	ID      string              `json:"id"`
	Glosses map[string]*Term    `json:"glosses"`
	Claims  map[string][]*Claim `json:"claims"`
}

type Sitelink struct {
	Site   string   `json:"site"`
	Title  string   `json:"title"`
	Badges []string `json:"badges,omitempty"`
	URL    string   `json:"url"`
}

type Term struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type Reference struct {
	Hash       string             `json:"hash"`
	Snaks      map[string][]*Snak `json:"snaks"`
	SnaksOrder []string           `json:"snaks-order"`
}

type Redirect struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ResponseError struct {
	Code  string `json:"code"`
	Info  string `json:"info"`
	Extra string `json:"extra"`
}

func (re *ResponseError) Error() string {
	return re.Info
}

type SimpleProperty struct {
	DataType     DataType                  `json:"type"`
	Labels       map[string]string         `json:"labels,omitempty"`
	Descriptions map[string]string         `json:"descriptions,omitempty"`
	Aliases      map[string][]string       `json:"aliases,omitempty"`
	Claims       map[string][]*SimpleClaim `json:"claims,omitempty"`
}

type SimpleItem struct {
	Labels       map[string]string         `json:"labels,omitempty"`
	Descriptions map[string]string         `json:"descriptions,omitempty"`
	Aliases      map[string][]string       `json:"aliases,omitempty"`
	Claims       map[string][]*SimpleClaim `json:"claims,omitempty"`
	Sitelinks    map[string]string         `json:"sitelinks,omitempty"`
}

type SimpleLexeme struct {
	DataType        DataType          `json:"type,omitempty"`
	LexicalCategory string            `json:"category,omitempty"`
	Language        string            `json:"language,omitempty"`
	Lemmas          map[string]string `json:"lemmas,omitempty"`
	Forms           []*SimpleForm     `json:"forms,omitempty"`
	Senses          []*SimpleSense    `json:"senses,omitempty"`
}

type SimpleForm struct {
	Representations     map[string]string         `json:"representations,omitempty"`
	GrammaticalFeatures []string                  `json:"features,omitempty"`
	Claims              map[string][]*SimpleClaim `json:"claims,omitempty"`
}

type SimpleSense struct {
	Glosses map[string]string         `json:"glosses,omitempty"`
	Claims  map[string][]*SimpleClaim `json:"claims,omitempty"`
}

type SimpleClaim struct {
	Type       DataType                `json:"type,omitempty"`
	Value      interface{}             `json:"value,omitempty"`
	Qualifiers map[string][]*SnakValue `json:"qualifiers,omitempty"`
}
