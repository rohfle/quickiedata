package quickiedata

import (
	"encoding/json"
)

type Claim struct {
	ID              string             `json:"id"`
	MainSnak        *Snak              `json:"mainsnak"`
	Rank            Rank               `json:"rank"`
	Type            string             `json:"type"`
	Qualifiers      map[string][]*Snak `json:"qualifiers"`
	QualifiersOrder []string           `json:"qualifiersOrder"`
	References      []*Reference       `json:"references"`
}

type SimpleClaim struct {
	Type       string                        `json:"t,omitempty"`
	Value      interface{}                   `json:"v,omitempty"`
	Qualifiers map[string][]*SimpleSnakValue `json:"q,omitempty"`
}

func (sc *SimpleClaim) UnmarshalJSON(data []byte) error {
	var peek struct {
		Type       string                        `json:"t,omitempty"`
		Value      json.RawMessage               `json:"v,omitempty"`
		Qualifiers map[string][]*SimpleSnakValue `json:"q,omitempty"`
	}

	err := json.Unmarshal(data, &peek)
	if err != nil {
		return err
	}

	sc.Type = peek.Type
	sc.Qualifiers = peek.Qualifiers

	value, err := unmarshalSimpleSnakValue(peek.Type, peek.Value)
	if err != nil {
		return err
	}
	sc.Value = value
	return nil
}

func (sc *SimpleClaim) GetQualifiers(key string) []*SimpleSnakValue {
	if sc == nil {
		return nil
	}

	qualifiers, ok := sc.Qualifiers[key]
	if !ok {
		return nil
	}

	return qualifiers
}

func (sc *SimpleClaim) GetQualifier(key string) *SimpleSnakValue {
	if sc == nil {
		return nil
	}

	qualifiers := sc.GetQualifiers(key)
	if len(qualifiers) == 0 {
		return nil
	}
	return qualifiers[0]
}

func (sc *SimpleClaim) ValueAsString() *string {
	if sc == nil {
		return nil
	}
	value, ok := sc.Value.(string)
	if !ok {
		return nil
	}
	return &value
}

func (sc *SimpleClaim) ValueAsCoordinate() *SnakValueGlobeCoordinate {
	if sc == nil {
		return nil
	}
	value, ok := sc.Value.(*SnakValueGlobeCoordinate)
	if !ok {
		return nil
	}
	return value
}

func (sc *SimpleClaim) ValueAsMonolingualText() *SnakValueMonolingualText {
	if sc == nil {
		return nil
	}
	value, ok := sc.Value.(*SnakValueMonolingualText)
	if !ok {
		return nil
	}
	return value
}

func (sc *SimpleClaim) ValueAsTime() *SnakValueTime {
	if sc == nil {
		return nil
	}
	value, ok := sc.Value.(*SnakValueTime)
	if !ok {
		return nil
	}
	return value
}

func (sc *SimpleClaim) ValueAsQuantity() *SnakValueQuantity {
	if sc == nil {
		return nil
	}
	value, ok := sc.Value.(*SnakValueQuantity)
	if !ok {
		return nil
	}
	return value
}

func (sc *SimpleClaim) ValueAsEntity() *SnakValueEntity {
	if sc == nil {
		return nil
	}
	value, ok := sc.Value.(*SnakValueEntity)
	if !ok {
		return nil
	}
	return value
}
