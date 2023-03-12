package quickiedata

import (
	"fmt"
	"strconv"
	"strings"
)

func SimplifyMapOfTermArray(terms map[string][]*Term) map[string][]string {
	var output = make(map[string][]string)
	for key, values := range terms {
		var valuesOut []string
		for _, value := range values {
			valuesOut = append(valuesOut, value.Value)
		}
		output[key] = valuesOut
	}
	return output
}

func SimplifyMapOfTerms(terms map[string]*Term) map[string]string {
	var output = make(map[string]string)
	for _, value := range terms {
		output[value.Language] = value.Value
	}
	return output
}

func SimplifyEntity(entity *EntityInfo) interface{} {
	switch entity.Type {
	case "item":
		return &SimpleItem{
			Labels:       SimplifyMapOfTerms(entity.Labels),
			Descriptions: SimplifyMapOfTerms(entity.Descriptions),
			Aliases:      SimplifyMapOfTermArray(entity.Aliases),
			Claims:       SimplifyClaims(entity.Claims),
			Sitelinks:    SimplifySitelinks(entity.Sitelinks),
		}
	case "property":
		return &SimpleProperty{
			DataType:     entity.DataType, // needed?
			Labels:       SimplifyMapOfTerms(entity.Labels),
			Descriptions: SimplifyMapOfTerms(entity.Descriptions),
			Aliases:      SimplifyMapOfTermArray(entity.Aliases),
			Claims:       SimplifyClaims(entity.Claims),
		}
	case "lexeme":
		return &SimpleLexeme{
			DataType:        entity.DataType, // needed?
			LexicalCategory: entity.LexicalCategory,
			Language:        entity.Language,
			Lemmas:          SimplifyMapOfTerms(entity.Lemmas),
			Forms:           SimplifyForms(entity.Forms),
			Senses:          SimplifySenses(entity.Senses),
		}
	case "form":
		return &SimpleForm{
			GrammaticalFeatures: entity.GrammaticalFeatures,
			Representations:     SimplifyMapOfTerms(entity.Representations),
			Claims:              SimplifyClaims(entity.Claims),
		}
	case "sense":
		return &SimpleSense{
			Glosses: SimplifyMapOfTerms(entity.Glosses),
			Claims:  SimplifyClaims(entity.Claims),
		}
	default:
		return nil
	}
}

func SimplifySitelinks(sitelinks map[string]*Sitelink) map[string]string {
	var output = make(map[string]string)
	for _, value := range sitelinks {
		output[value.Site] = value.Title
	}
	return output
}

func SimplifySenses(senses []*Sense) []*SimpleSense {
	var output []*SimpleSense
	for _, sense := range senses {
		output = append(output, &SimpleSense{
			Glosses: SimplifyMapOfTerms(sense.Glosses),
			Claims:  SimplifyClaims(sense.Claims),
		})
	}
	return output
}

func SimplifyForms(forms []*Form) []*SimpleForm {
	var output []*SimpleForm
	for _, form := range forms {
		output = append(output, &SimpleForm{
			Representations:     SimplifyMapOfTerms(form.Representations),
			GrammaticalFeatures: form.GrammaticalFeatures,
			Claims:              SimplifyClaims(form.Claims),
		})
	}
	return output
}

func SimplifyClaims(claimMap map[string][]*Claim) map[string][]*SimpleClaim {
	var output = make(map[string][]*SimpleClaim)

	for key, claims := range claimMap {
		var newClaims []*SimpleClaim
		for _, claim := range claims {
			mainSnak := SimplifySnak(claim.MainSnak)
			if mainSnak == nil {
				continue
			}
			simpleClaim := &SimpleClaim{
				Type:  mainSnak.Type,
				Value: mainSnak.Value,
			}
			if len(claim.Qualifiers) > 0 {
				simpleClaim.Qualifiers = SimplifySnaks(claim.Qualifiers)
			}
			newClaims = append(newClaims, simpleClaim)
		}
		if len(newClaims) > 0 {
			output[key] = newClaims
		}
	}
	return output
}

func SimplifySnak(snak *Snak) *SnakValue {
	if snak.SnakType != "value" {
		return nil
	}

	stype := snak.DataType
	if stype == "" {
		stype = snak.DataValue.Type
	}

	return &SnakValue{
		Type:  stype,
		Value: ParseClaim(snak.DataValue),
	}
}

func SimplifySnaks(snakMap map[string][]*Snak) map[string][]*SnakValue {
	var output = make(map[string][]*SnakValue)

	for key, snaks := range snakMap {
		var newSnaks []*SnakValue
		for _, snak := range snaks {
			if snak.SnakType == "value" {
				newSnaks = append(newSnaks, SimplifySnak(snak))
			}
		}
		if len(newSnaks) > 0 {
			output[key] = newSnaks
		}
	}
	return output
}

func ParseClaim(dv *SnakValue) interface{} {
	switch value := dv.Value.(type) {
	case string:
		return value
	case *SnakValueEntity:
		return value.ID
	case *SnakValueMonolingualText:
		return value.Value
	case *SnakValueGlobeCoordinate:
		// convert globe
		return &SnakValueGlobeCoordinate{
			Latitude:  value.Latitude,
			Longitude: value.Longitude,
			Altitude:  value.Altitude,
			Precision: value.Precision,
			Globe:     GetWikidataIDFromURL(value.Globe),
		}
	case *SnakValueQuantity:
		// use blank string for no unit
		unit := ""
		if value.Unit != "1" {
			unit = value.Unit
		}
		return &SnakValueQuantity{
			Amount:     value.Amount,
			Unit:       GetWikidataIDFromURL(unit),
			UpperBound: value.UpperBound,
			LowerBound: value.LowerBound,
		}
	case *SnakValueTime:
		return &SnakValueTime{
			After:         value.After,
			Before:        value.Before,
			CalendarModel: GetWikidataIDFromURL(value.CalendarModel),
			Precision:     value.Precision,
			// Fix month / date in wikidata returned date strings
			// Notes: may still not be valid ISO strings as years might be very big
			Time:     strings.ReplaceAll(value.Time, "-00", "-01"),
			Timezone: value.Timezone,
		}
	default:
		// unexpected datatype
		// TODO: warn?
		return value
	}
}

func SimplifySPARQLDataType(s string) string {
	return strings.ToLower(strings.TrimPrefix(s, "http://www.w3.org/2001/XMLSchema#"))
}

func SimplifyWikidataURI(uri string) (string, error) {
	s := strings.TrimPrefix(uri, "http://www.wikidata.org/")
	if strings.HasPrefix(s, "entity/statement/") {
		s = strings.TrimPrefix(s, "entity/statement/")
		return strings.Replace(s, "-", "$", 1), nil
	} else if strings.HasPrefix(s, "entity/") {
		return strings.TrimPrefix(s, "entity/"), nil
	} else if strings.HasPrefix(s, "prop/direct/") {
		return strings.TrimPrefix(s, "prop/direct/"), nil
	}
	// if unsure return original
	return uri, nil
}

func SimplifyBindingValue(bvalue *BindingValue) (*SimpleBindingValue, error) {
	switch bvalue.Type {
	case "uri":
		if bvalue.Value == nil {
			return nil, nil
		}
		value, err := SimplifyWikidataURI(*bvalue.Value)
		if err != nil {
			return nil, err
		}
		return &SimpleBindingValue{
			Value: value,
		}, nil
	case "bnode":
		return nil, nil
	case "literal":
		if bvalue.Value == nil {
			return nil, nil
		}
		datatype := SimplifySPARQLDataType(bvalue.DataType)
		switch datatype {
		case "boolean":
			return &SimpleBindingValue{
				Value: *bvalue.Value == "true",
			}, nil
		case "integer":
			value, err := strconv.ParseInt(*bvalue.Value, 10, 64)
			if err != nil {
				return nil, err
			}
			return &SimpleBindingValue{
				Value: value,
			}, nil
		case "float":
			value, err := strconv.ParseFloat(*bvalue.Value, 64)
			if err != nil {
				return nil, err
			}
			return &SimpleBindingValue{
				Value: value,
			}, nil
		default: // including unknown types, string, datetime
			return &SimpleBindingValue{
				Value: *bvalue.Value,
			}, nil
		}
	default:
		return nil, fmt.Errorf("unknown type '%s'", bvalue.Type)
	}
}
