package quickiedata_test

import (
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

		var compareEntity interface{}

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

			if t.Failed() {
				t.FailNow()
			}
		}
	}

}

func getTestEntitySimplifyCouples(baseDir string) ([][2]string, error) {
	var couples map[string][2]string = make(map[string][2]string)

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
