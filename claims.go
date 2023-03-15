package quickiedata

type Claim struct {
	ID              string             `json:"id"`
	MainSnak        *Snak              `json:"mainsnak"`
	Rank            Rank               `json:"rank"`
	Type            DataType           `json:"type"`
	Qualifiers      map[string][]*Snak `json:"qualifiers"`
	QualifiersOrder []string           `json:"qualifiersOrder"`
	References      []*Reference       `json:"references"`
}

type SimpleClaim struct {
	Type       DataType                `json:"type,omitempty"`
	Value      interface{}             `json:"value,omitempty"`
	Qualifiers map[string][]*SnakValue `json:"qualifiers,omitempty"`
}

func (sc *SimpleClaim) GetQualifiers(key string) []*SnakValue {
	if sc == nil {
		return nil
	}

	qualifiers, ok := sc.Qualifiers[key]
	if !ok {
		return nil
	}

	return qualifiers
}

func (sc *SimpleClaim) GetQualifier(key string) *SnakValue {
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
