package helpers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rohfle/quickiedata/types"
)

func SimplifyMapOfTermArray(terms map[string][]*types.Term) map[string][]string {
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

func SimplifyMapOfTerms(terms map[string]*types.Term) map[string]string {
	var output = make(map[string]string)
	for _, value := range terms {
		output[value.Language] = value.Value
	}
	return output
}

func SimplifyEntities(entities map[string]*types.EntityInfo) map[string]interface{} {
	var output = make(map[string]interface{})
	for key, entity := range entities {
		switch entity.Type {
		case "item":
			output[key] = &types.SimpleItem{
				Labels:       SimplifyMapOfTerms(entity.Labels),
				Descriptions: SimplifyMapOfTerms(entity.Descriptions),
				Aliases:      SimplifyMapOfTermArray(entity.Aliases),
				Claims:       SimplifyClaims(entity.Claims),
				Sitelinks:    SimplifySitelinks(entity.Sitelinks),
			}
		case "property":
			output[key] = &types.SimpleProperty{
				DataType:     entity.DataType, // needed?
				Labels:       SimplifyMapOfTerms(entity.Labels),
				Descriptions: SimplifyMapOfTerms(entity.Descriptions),
				Aliases:      SimplifyMapOfTermArray(entity.Aliases),
				Claims:       SimplifyClaims(entity.Claims),
			}
		case "lexeme":
			output[key] = &types.SimpleLexeme{
				DataType:        entity.DataType, // needed?
				LexicalCategory: entity.LexicalCategory,
				Language:        entity.Language,
				Lemmas:          SimplifyMapOfTerms(entity.Lemmas),
				Forms:           SimplifyForms(entity.Forms),
				Senses:          SimplifySenses(entity.Senses),
			}
		case "form":
			output[key] = &types.SimpleForm{
				GrammaticalFeatures: entity.GrammaticalFeatures,
				Representations:     SimplifyMapOfTerms(entity.Representations),
				Claims:              SimplifyClaims(entity.Claims),
			}
		case "sense":
			output[key] = &types.SimpleSense{
				Glosses: SimplifyMapOfTerms(entity.Glosses),
				Claims:  SimplifyClaims(entity.Claims),
			}
		}
	}
	return output
}

func SimplifySitelinks(sitelinks map[string]*types.Sitelink) map[string]string {
	var output = make(map[string]string)
	for _, value := range sitelinks {
		output[value.Site] = value.Title
	}
	return output
}

func SimplifySenses(senses []*types.Sense) []*types.SimpleSense {
	var output []*types.SimpleSense
	for _, sense := range senses {
		output = append(output, &types.SimpleSense{
			Glosses: SimplifyMapOfTerms(sense.Glosses),
			Claims:  SimplifyClaims(sense.Claims),
		})
	}
	return output
}

func SimplifyForms(forms []*types.Form) []*types.SimpleForm {
	var output []*types.SimpleForm
	for _, form := range forms {
		output = append(output, &types.SimpleForm{
			Representations:     SimplifyMapOfTerms(form.Representations),
			GrammaticalFeatures: form.GrammaticalFeatures,
			Claims:              SimplifyClaims(form.Claims),
		})
	}
	return output
}

func SimplifyClaims(claimMap map[string][]*types.Claim) map[string][]*types.SimpleClaim {
	var output = make(map[string][]*types.SimpleClaim)

	for key, claims := range claimMap {
		var newClaims []*types.SimpleClaim
		for _, claim := range claims {
			mainSnak := SimplifySnak(claim.MainSnak)
			if mainSnak == nil {
				continue
			}
			simpleClaim := &types.SimpleClaim{
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

func SimplifySnak(snak *types.Snak) *types.SnakValue {
	if snak.SnakType != "value" {
		return nil
	}

	stype := snak.DataType
	if stype == "" {
		stype = snak.DataValue.Type
	}

	return &types.SnakValue{
		Type:  stype,
		Value: ParseClaim(snak.DataValue),
	}
}

func SimplifySnaks(snakMap map[string][]*types.Snak) map[string][]*types.SnakValue {
	var output = make(map[string][]*types.SnakValue)

	for key, snaks := range snakMap {
		var newSnaks []*types.SnakValue
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

func ParseClaim(dv *types.SnakValue) interface{} {
	switch value := dv.Value.(type) {
	case string:
		return value
	case *types.SnakValueEntity:
		return value.ID
	case *types.SnakValueMonolingualText:
		return value.Value
	case *types.SnakValueGlobeCoordinate:
		// convert globe
		return &types.SnakValueGlobeCoordinate{
			Latitude:  value.Latitude,
			Longitude: value.Longitude,
			Altitude:  value.Altitude,
			Precision: value.Precision,
			Globe:     GetWikidataIDFromURL(value.Globe),
		}
	case *types.SnakValueQuantity:
		// use blank string for no unit
		unit := ""
		if value.Unit != "1" {
			unit = value.Unit
		}
		return &types.SnakValueQuantity{
			Amount:     value.Amount,
			Unit:       GetWikidataIDFromURL(unit),
			UpperBound: value.UpperBound,
			LowerBound: value.LowerBound,
		}
	case *types.SnakValueTime:
		return &types.SnakValueTime{
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

func GetWikidataIDFromURL(url string) string {
	if url == "" {
		return ""
	}
	return strings.TrimPrefix(url, "http://www.wikidata.org/entity/")
}

func SimplifySPARQLResults(results *types.SPARQLResults) []map[string]interface{} {
	var output []map[string]interface{}
	for _, binding := range results.Results.Bindings {
		var newResult = make(map[string]interface{})
		for key, bvalue := range binding {
			val, err := SimplifyBindingValue(bvalue)
			if err != nil {
				if bvalue.Value != nil {
					fmt.Printf("error while simplifying %s value %v: %s\n", bvalue.DataType, *bvalue.Value, err)
				} else {
					fmt.Printf("error while simplifying %s value <nil>: %s\n", bvalue.DataType, err)
				}
				continue
			}
			newResult[key] = val
		}
		output = append(output, newResult)
	}
	return output
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

func SimplifyBindingValue(bvalue *types.BindingValue) (interface{}, error) {
	switch bvalue.Type {
	case "uri":
		if bvalue.Value == nil {
			return nil, nil
		}
		return SimplifyWikidataURI(*bvalue.Value)
	case "bnode":
		return nil, nil
	case "literal":
		if bvalue.Value == nil {
			return nil, nil
		}
		datatype := SimplifySPARQLDataType(bvalue.DataType)
		switch datatype {
		case "string", "datetime":
			return *bvalue.Value, nil
		case "boolean":
			return *bvalue.Value == "true", nil
		case "integer":
			return strconv.ParseInt(*bvalue.Value, 10, 64)
		case "float":
			return strconv.ParseFloat(*bvalue.Value, 64)
		default:
			// return unknown type as string
			return *bvalue.Value, nil
		}
	default:
		return nil, fmt.Errorf("unknown type '%s'", bvalue.Type)
	}
}
