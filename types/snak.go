package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
