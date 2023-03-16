package quickiedata

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Snak struct {
	ID        string     `json:"id"`
	DataType  string     `json:"datatype"`
	DataValue *SnakValue `json:"datavalue"`
	Hash      string     `json:"hash"`
	Property  string     `json:"property"`
	SnakType  string     `json:"snaktype"`
}

type SnakValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (sv *SnakValue) ValueAsString() *string {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*string)
	return value
}

func (sv *SnakValue) ValueAsCoordinate() *SnakValueGlobeCoordinate {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*SnakValueGlobeCoordinate)
	return value
}

func (sv *SnakValue) ValueAsMonolingualText() *SnakValueMonolingualText {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*SnakValueMonolingualText)
	return value
}

func (sv *SnakValue) ValueAsTime() *SnakValueTime {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*SnakValueTime)
	return value
}

func (sv *SnakValue) ValueAsQuantity() *SnakValueQuantity {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*SnakValueQuantity)
	return value
}

func (sv *SnakValue) ValueAsEntity() *SnakValueEntity {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*SnakValueEntity)
	return value
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

func (s *SnakValueTime) GetYear() *int {
	if s == nil {
		return nil
	}
	yearStr := strings.SplitN(strings.TrimPrefix(s.Time, "+"), "-", 2)[0]
	if val, err := strconv.Atoi(yearStr); err == nil {
		return &val
	}
	// its possible that the year might not fit in an int32
	// so might need to process the field yourself
	return nil
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
	sv.Type = peek.Type

	workingType := sv.Type
	if DataTypeIsSimple(sv.Type) {
		workingType = "simple"
	} else if DataTypeIsEntity(sv.Type) {
		workingType = "entity"
	}

	switch workingType {
	case "simple":
		s := ""
		sv.Value = &s
	case "entity":
		sv.Value = &SnakValueEntity{}
	case "globecoordinate", "globe-coordinate":
		sv.Value = &SnakValueGlobeCoordinate{}
	case "monolingualtext":
		sv.Value = &SnakValueMonolingualText{}
	case "quantity":
		sv.Value = &SnakValueQuantity{}
	case "time":
		sv.Value = &SnakValueTime{}
	default:
		return fmt.Errorf("%s snak value parser not implemented", workingType)
	}

	return json.Unmarshal(peek.Value, sv.Value)
}

type SimpleSnakValue struct {
	Type  string      `json:"t"`
	Value interface{} `json:"v"`
}

func (sv *SimpleSnakValue) ValueAsString() *string {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*string)
	return value
}

func (sv *SimpleSnakValue) ValueAsCoordinate() *SnakValueGlobeCoordinate {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*SnakValueGlobeCoordinate)
	return value
}

func (sv *SimpleSnakValue) ValueAsTime() *SnakValueTime {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*SnakValueTime)
	return value
}

func (sv *SimpleSnakValue) ValueAsQuantity() *SnakValueQuantity {
	if sv == nil {
		return nil
	}
	value, _ := sv.Value.(*SnakValueQuantity)
	return value
}

func (sv *SimpleSnakValue) UnmarshalJSON(data []byte) error {
	var peek struct {
		Type  string          `json:"t"`
		Value json.RawMessage `json:"v"`
	}
	err := json.Unmarshal(data, &peek)
	if err != nil {
		return err
	}

	sv.Type = peek.Type
	value, err := unmarshalSimpleSnakValue(string(peek.Type), peek.Value)
	if err != nil {
		return err
	}
	sv.Value = value
	return nil
}

func unmarshalSimpleSnakValue(stype string, data []byte) (interface{}, error) {
	var value interface{}

	switch stype {
	case "string", "external", "item", "url", "property", "lexeme", "media", "geoshape", "musical":
		s := ""
		value = &s
	case "quantity":
		value = &SnakValueQuantity{}
	case "coords":
		value = &SnakValueGlobeCoordinate{}
	case "time":
		value = &SnakValueTime{}
	default:
		err := fmt.Errorf("%s simple snak value parser not implemented: %s", stype, string(data))
		return nil, err
	}

	err := json.Unmarshal(data, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}
