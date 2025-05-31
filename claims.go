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
	Type       string                        `json:"type,omitempty"`
	Rank       string                        `json:"rank,omitempty"`
	Value      any                           `json:"value,omitempty"`
	Qualifiers map[string][]*SimpleSnakValue `json:"qualifiers,omitempty"`
}

func (sc *SimpleClaim) UnmarshalJSON(data []byte) error {
	var peek struct {
		Type       string                        `json:"type,omitempty"`
		Rank       string                        `json:"rank,omitempty"`
		Value      json.RawMessage               `json:"value,omitempty"`
		Qualifiers map[string][]*SimpleSnakValue `json:"qualifiers,omitempty"`
	}

	err := json.Unmarshal(data, &peek)
	if err != nil {
		return err
	}

	sc.Type = peek.Type
	sc.Rank = peek.Rank
	sc.Qualifiers = peek.Qualifiers

	sc.Value, err = unmarshalSimpleSnakValue(peek.Type, peek.Value)
	return err
}

func (sc *SimpleClaim) GetQualifiers(key string) []*SimpleSnakValue {
	if sc == nil {
		return nil
	}

	return sc.Qualifiers[key]
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
	value, _ := sc.Value.(*string)
	return value
}

func (sc *SimpleClaim) ValueAsCoordinate() *SnakValueGlobeCoordinate {
	if sc == nil {
		return nil
	}
	value, _ := sc.Value.(*SnakValueGlobeCoordinate)
	return value
}

func (sc *SimpleClaim) ValueAsMonolingualText() *SnakValueMonolingualText {
	if sc == nil {
		return nil
	}
	value, _ := sc.Value.(*SnakValueMonolingualText)
	return value
}

func (sc *SimpleClaim) ValueAsTime() *SnakValueTime {
	if sc == nil {
		return nil
	}
	value, _ := sc.Value.(*SnakValueTime)
	return value
}

func (sc *SimpleClaim) ValueAsQuantity() *SnakValueQuantity {
	if sc == nil {
		return nil
	}
	value, _ := sc.Value.(*SnakValueQuantity)
	return value
}

func (sc *SimpleClaim) ValueAsEntity() *SnakValueEntity {
	if sc == nil {
		return nil
	}
	value, _ := sc.Value.(*SnakValueEntity)
	return value
}
