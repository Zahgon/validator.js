package validator_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	validator "github.com/validatorjs/validator.js"
	"github.com/validatorjs/validator.js/internal/wire"
	"github.com/validatorjs/validator.js/js"
)

// fixtureCase is one recorded call of the original test suite: the export, its arguments and either the
// returned value or the thrown error, in the tagged JSON form of the validator command.
type fixtureCase struct {
	It    string          `json:"it"`
	Fn    string          `json:"fn"`
	Args  json.RawMessage `json:"args"`
	OK    json.RawMessage `json:"ok"`
	Error *struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"error"`
}

type fixture struct {
	File  string        `json:"file"`
	Cases []fixtureCase `json:"cases"`
}

func loadFixture(t *testing.T, name string) fixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var f fixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return f
}

func jsonEqual(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// checkCase calls the export and compares the outcome with the recorded one.
func checkCase(t *testing.T, c fixtureCase) {
	t.Helper()
	args, err := wire.DecodeArgs(c.Args)
	if err != nil {
		t.Fatalf("%s(%s): bad fixture arguments: %v", c.Fn, c.Args, err)
	}
	got, callErr := validator.Call(c.Fn, args)
	if c.Error != nil {
		jsErr, ok := callErr.(*js.Error)
		if !ok || jsErr.Name != c.Error.Name || jsErr.Message != c.Error.Message {
			t.Errorf("%s(%s) = %v, %v; want %s: %s", c.Fn, c.Args, got, callErr, c.Error.Name, c.Error.Message)
		}
		return
	}
	if callErr != nil {
		t.Errorf("%s(%s) threw %v; want %s", c.Fn, c.Args, callErr, c.OK)
		return
	}
	var buf bytes.Buffer
	wire.Encode(&buf, got)
	if !jsonEqual(buf.Bytes(), c.OK) {
		t.Errorf("%s(%s) = %s; want %s", c.Fn, c.Args, buf.Bytes(), c.OK)
	}
}

// runFixture runs every recorded call of one upstream test file, one subtest per original `it` block.
func runFixture(t *testing.T, name string) {
	f := loadFixture(t, name)
	var order []string
	byIt := map[string][]fixtureCase{}
	for _, c := range f.Cases {
		if _, seen := byIt[c.It]; !seen {
			order = append(order, c.It)
		}
		byIt[c.It] = append(byIt[c.It], c)
	}
	for _, it := range order {
		cases := byIt[it]
		t.Run(it, func(t *testing.T) {
			for _, c := range cases {
				checkCase(t, c)
			}
		})
	}
}
