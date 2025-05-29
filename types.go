package quickiedata

import (
	"encoding/json"
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

// DataType is the Wikidata datatype
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

// MapSimplifyType is a lookup table used to simplify data types
var MapSimplifyType = map[string]string{
	"commonsMedia":      "media",
	"external-id":       "external",
	"geo-shape":         "geoshape",
	"globecoordinate":   "coords",
	"globe-coordinate":  "coords",
	"monolingualtext":   "string",
	"musical-notation":  "musical",
	"tabular-data":      "tabular",
	"wikibase-entityid": "string",
	"wikibase-form":     "form",
	"wikibase-item":     "item",
	"wikibase-lexeme":   "lexeme",
	"wikibase-property": "property",
	"wikibase-sense":    "sense",
}

// Internal aliases that simplify matters, but aren't used by wikidata
const (
	DataTypeSimple DataType = "simple"
	DataTypeEntity DataType = "entity"
)

var ListOfSimpleDataTypes = []DataType{
	DataTypeCommonsMedia,
	DataTypeExternalID,
	DataTypeGeoShape,
	DataTypeMath,
	DataTypeMusicalNotation,
	DataTypeString,
	DataTypeTabularData,
	DataTypeURL,
}

var ListOfEntityDataTypes = []DataType{
	DataTypeWikibaseEntityID,
	DataTypeWikibaseForm,
	DataTypeWikibaseItem,
	DataTypeWikibaseLexeme,
	DataTypeWikibaseProperty,
	DataTypeWikibaseSense,
}

func DataTypeIsSimple(dtype string) bool {
	return ValueInSlice(DataType(dtype), ListOfSimpleDataTypes)
}
func DataTypeIsEntity(dtype string) bool {
	return ValueInSlice(DataType(dtype), ListOfEntityDataTypes)
}

func ValueInSlice[T comparable](needle T, haystack []T) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}

// NumberPlus extends json.Number to removes the prefix "+" and prevent errors when unmarshaling
type NumberPlus json.Number

func (n *NumberPlus) Float64() (float64, error) {
	return json.Number(*n).Float64()
}
func (n *NumberPlus) Int64() (int64, error) {
	return json.Number(*n).Int64()
}
func (n *NumberPlus) String() string {
	return json.Number(*n).String()
}

func (n *NumberPlus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	str = strings.TrimPrefix(str, "+")
	*n = NumberPlus(str)
	return nil
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
