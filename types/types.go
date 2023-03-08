package types

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
