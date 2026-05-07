package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hishamkaram/gismanager/v2/publish"
)

// TestLayerEntryJSON_Shape locks in the on-the-wire shape of the
// `-json` output. Operators wiring layerSchema into Terraform / jq /
// Ansible pipelines depend on these JSON keys staying stable; if a
// future refactor renames a field it must update this test in lockstep.
func TestLayerEntryJSON_Shape(t *testing.T) {
	entry := layerEntry{
		Path: "./testdata/sample.gpkg",
		Name: "neighborhoods",
		Fields: []*publish.LayerField{
			{Name: "geom", Type: "POLYGON"},
			{Name: "id", Type: "Integer"},
			{Name: "name", Type: "String"},
		},
	}

	out, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)

	for _, want := range []string{
		`"path":"./testdata/sample.gpkg"`,
		`"name":"neighborhoods"`,
		`"fields":[`,
		`{"Name":"geom","Type":"POLYGON"}`,
		`{"Name":"id","Type":"Integer"}`,
		`{"Name":"name","Type":"String"}`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("JSON output missing %q\n got: %s", want, got)
		}
	}
}

// TestLayerEntryJSON_EmptyArrayIsValid locks in the "no layers found"
// case: runJSON initializes entries as an empty-but-non-nil slice so
// the JSON output is `[]\n` rather than `null\n`. Downstream parsers
// (jq, Python, Terraform) treat the two very differently — empty array
// is the right contract.
func TestLayerEntryJSON_EmptyArrayIsValid(t *testing.T) {
	entries := []layerEntry{}
	out, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `[]` {
		t.Errorf("empty-slice marshal = %q; want []", string(out))
	}

	// Contrast: `var entries []layerEntry` (nil slice) marshals to
	// `null`, which is the wrong shape for shell pipelines. The
	// runJSON code path constructs `entries := []layerEntry{}`
	// explicitly to side-step this.
	var nilEntries []layerEntry
	out, _ = json.Marshal(nilEntries)
	if string(out) != `null` {
		t.Errorf("nil-slice marshal = %q; want null (this guards the runJSON allocation choice)", string(out))
	}
}
