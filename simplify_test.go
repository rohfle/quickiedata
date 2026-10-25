package quickiedata_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/rohfle/quickiedata"
)

func TestEntitySimplify(t *testing.T) {
	couples, err := getTestEntitySimplifyCouples("testdata/simplify")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("found %d couples \n", len(couples))

	for _, testCouple := range couples {
		beforeData, err := os.ReadFile(testCouple[0])
		if err != nil {
			t.Fatal(err)
		}

		var beforeEntity quickiedata.EntityInfo
		err = json.Unmarshal(beforeData, &beforeEntity)
		if err != nil {
			t.Fatal(err)
		}

		afterEntity := quickiedata.SimplifyEntity(&beforeEntity)

		compareData, err := os.ReadFile(testCouple[1])
		if err != nil {
			t.Fatal(err)
		}

		// Useful widget to populate initial testdata
		if strings.HasPrefix(string(compareData), "{}") {
			t.Logf("found special symbol %q in %q. populating expected simpified entity data...", "{}", testCouple[1])
			afterdata, _ := json.MarshalIndent(afterEntity, "", "  ")
			os.WriteFile(testCouple[1], afterdata, 0644)
			continue
		}

		var compareEntity any

		wikidataID := strings.SplitN(path.Base(testCouple[0]), ".", 2)[0]
		switch wikidataID[0] {
		case 'Q':
			var temp quickiedata.SimpleItem
			err = json.Unmarshal(compareData, &temp)
			if err != nil {
				t.Fatal(err)
			}
			compareEntity = &temp
		case 'P':
			var temp quickiedata.SimpleProperty
			err = json.Unmarshal(compareData, &temp)
			if err != nil {
				t.Fatal(err)
			}
			compareEntity = &temp
		case 'L':
			var temp quickiedata.SimpleLexeme
			err = json.Unmarshal(compareData, &temp)
			if err != nil {
				t.Fatal(err)
			}
			compareEntity = &temp
			// this will error on Form and Sense
		default:
			t.Fatal("no handler", testCouple)
		}

		if diff := deep.Equal(afterEntity, compareEntity); diff != nil {
			t.Error("unmatching entities for", testCouple[0], "->", testCouple[1])

			for _, line := range diff {
				t.Error(line)
			}
		} else {
			t.Log(testCouple[0], "->", testCouple[1], "matches!")
		}
	}
	if t.Failed() {
		t.FailNow()
	}

}

func getTestEntitySimplifyCouples(baseDir string) ([][2]string, error) {
	var couples = make(map[string][2]string)

	files, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	// find matching .json / .simple.json pairs
	for _, f := range files {
		name := f.Name()
		bits := strings.SplitN(name, ".", 2)
		if len(bits) != 2 {
			continue
		}

		couple := couples[bits[0]]

		switch bits[1] {
		case "json":
			couple[0] = path.Join(baseDir, name)
		case "simple.json":
			couple[1] = path.Join(baseDir, name)
		default:
			continue
		}

		couples[bits[0]] = couple
	}

	// only return matching pairs
	var toReturn = make([][2]string, 0, len(couples))
	for _, couple := range couples {
		if couple[0] != "" && couple[1] != "" {
			toReturn = append(toReturn, couple)
		}
	}

	return toReturn, nil
}

// A null snak must not stop the simplification of the claims around it.
func TestSimplifyEntityWithNullSnaks(t *testing.T) {
	raw := `{"id":"Q1","type":"item","claims":{
		"P1":[{"mainsnak":null,"rank":"normal"}],
		"P2":[{"rank":"normal",
			"mainsnak":{"snaktype":"value","datatype":"string","datavalue":{"value":"kept","type":"string"}},
			"qualifiers":{"P3":[null]}}]
	}}`

	var entity quickiedata.EntityInfo
	if err := json.Unmarshal([]byte(raw), &entity); err != nil {
		t.Fatal(err)
	}

	simplified := quickiedata.SimplifyEntity(&entity)
	item, ok := simplified.(*quickiedata.SimpleItem)
	if !ok {
		t.Fatalf("expected a SimpleItem, got %T", simplified)
	}

	if claim := item.GetClaim("P1"); claim != nil {
		t.Errorf("a claim with a null snak must be dropped, got %+v", claim)
	}
	if got := item.GetClaim("P2").ValueAsString(); got == nil || *got != "kept" {
		t.Errorf("P2 must be kept, got %+v", got)
	}
	if qualifier := item.GetClaim("P2").GetQualifier("P3"); qualifier != nil {
		t.Errorf("a null qualifier must be dropped, got %+v", qualifier)
	}
	if got := quickiedata.SimplifyEntity(nil); got != nil {
		t.Errorf("SimplifyEntity(nil) = %+v, want nil", got)
	}
}

// Every datatype the simplifier can emit must survive a trip through the
// simple json form, or the round trip fails on a type it produced itself.
func TestSimpleFormRoundTripsEveryDataType(t *testing.T) {
	claims := []struct {
		property string
		snak     string // json of one mainsnak
	}{
		{"P1", `{"snaktype":"value","datatype":"string","datavalue":{"value":"text","type":"string"}}`},
		{"P2", `{"snaktype":"value","datatype":"url","datavalue":{"value":"https://example.org/x","type":"string"}}`},
		{"P3", `{"snaktype":"value","datatype":"math","datavalue":{"value":"E=mc^2","type":"string"}}`},
		{"P4", `{"snaktype":"value","datatype":"external-id","datavalue":{"value":"abc-123","type":"string"}}`},
		{"P5", `{"snaktype":"value","datatype":"commonsMedia","datavalue":{"value":"Foo.jpg","type":"string"}}`},
		{"P6", `{"snaktype":"value","datatype":"geo-shape","datavalue":{"value":"Data:Foo.map","type":"string"}}`},
		{"P7", `{"snaktype":"value","datatype":"musical-notation","datavalue":{"value":"\\clef treble","type":"string"}}`},
		{"P8", `{"snaktype":"value","datatype":"tabular-data","datavalue":{"value":"Data:Foo.tab","type":"string"}}`},
		{"P9", `{"snaktype":"value","datatype":"monolingualtext","datavalue":{"value":{"text":"hallo","language":"de"},"type":"monolingualtext"}}`},
		{"P10", `{"snaktype":"value","datatype":"quantity","datavalue":{"value":{"amount":"+1","unit":"1"},"type":"quantity"}}`},
		{"P11", `{"snaktype":"value","datatype":"time","datavalue":{"value":{"time":"+2020-01-01T00:00:00Z","timezone":0,"before":0,"after":0,"precision":11,"calendarmodel":"http://www.wikidata.org/entity/Q1985727"},"type":"time"}}`},
		{"P12", `{"snaktype":"value","datatype":"globe-coordinate","datavalue":{"value":{"latitude":52,"longitude":8,"precision":0.01,"globe":"http://www.wikidata.org/entity/Q2"},"type":"globecoordinate"}}`},
		{"P13", `{"snaktype":"value","datatype":"wikibase-item","datavalue":{"value":{"id":"Q42","entity-type":"item","numeric-id":42},"type":"wikibase-entityid"}}`},
		{"P14", `{"snaktype":"value","datatype":"wikibase-form","datavalue":{"value":{"id":"L1-F1","entity-type":"form","numeric-id":1},"type":"wikibase-entityid"}}`},
		{"P15", `{"snaktype":"value","datatype":"wikibase-sense","datavalue":{"value":{"id":"L1-S1","entity-type":"sense","numeric-id":1},"type":"wikibase-entityid"}}`},
	}

	var raw strings.Builder
	raw.WriteString(`{"id":"Q1","type":"item","claims":{`)
	for i, claim := range claims {
		if i > 0 {
			raw.WriteString(",")
		}
		fmt.Fprintf(&raw, `%q:[{"mainsnak":%s,"type":"statement","rank":"normal"}]`, claim.property, claim.snak)
	}
	raw.WriteString(`}}`)

	var entity quickiedata.EntityInfo
	if err := json.Unmarshal([]byte(raw.String()), &entity); err != nil {
		t.Fatalf("unmarshal entity: %s", err)
	}

	data, err := json.Marshal(quickiedata.SimplifyEntity(&entity))
	if err != nil {
		t.Fatalf("marshal simplified entity: %s", err)
	}

	var single quickiedata.GetEntitySimpleResponse
	if err := json.Unmarshal([]byte(`{"entity":`+string(data)+`}`), &single); err != nil {
		t.Fatalf("round trip through the single entity form failed: %s", err)
	}

	var multi quickiedata.GetEntitiesSimpleResponse
	if err := json.Unmarshal([]byte(`{"entities":{"Q1":`+string(data)+`}}`), &multi); err != nil {
		t.Fatalf("round trip through the entities form failed: %s", err)
	}

	// for the types that carry a plain string, the value must survive too
	wantValues := map[string]string{
		"P1":  "text",
		"P2":  "https://example.org/x",
		"P3":  "E=mc^2",
		"P4":  "abc-123",
		"P5":  "Foo.jpg",
		"P6":  "Data:Foo.map",
		"P7":  `\clef treble`,
		"P8":  "Data:Foo.tab",
		"P9":  "hallo",
		"P13": "Q42",
		"P14": "L1-F1",
		"P15": "L1-S1",
	}

	items := map[string]*quickiedata.SimpleItem{
		"single entity form": single.GetEntityAsItem(),
		"entities form":      multi.GetEntityAsItem("Q1"),
	}
	for name, item := range items {
		if item == nil {
			t.Fatalf("round trip through the %s lost the item: %s", name, data)
		}
		for _, claim := range claims {
			if item.GetClaim(claim.property) == nil {
				t.Errorf("round trip through the %s lost claim %s: %s", name, claim.property, data)
			}
		}
		for property, want := range wantValues {
			got := item.GetClaim(property).ValueAsString()
			if got == nil || *got != want {
				t.Errorf("round trip through the %s changed %s: got %v, want %q", name, property, got, want)
			}
		}
	}
}

// An unexpected datatype is a detail for a debugger, not something a library
// prints over the caller's standard error.
func TestUnexpectedDatatypeGoesToDebugLog(t *testing.T) {
	var logged bytes.Buffer
	previous := quickiedata.DebugLog.Writer()
	quickiedata.DebugLog.SetOutput(&logged)
	defer quickiedata.DebugLog.SetOutput(previous)

	// an int is not one of the datatypes this library knows
	snak := &quickiedata.SnakValue{Type: "unknown", Value: 42}
	if got := quickiedata.ParseClaim(snak); got != 42 {
		t.Errorf("ParseClaim = %v, want the value unchanged", got)
	}
	if !strings.Contains(logged.String(), "unexpected datatype") {
		t.Errorf("debug log = %q, want the unexpected datatype in it", logged.String())
	}
}
