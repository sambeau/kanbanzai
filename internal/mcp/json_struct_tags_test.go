package mcp

import (
	"reflect"
	"testing"

	"github.com/sambeau/kanbanzai/internal/docint"
)

// Audited structs (REQ-007, REQ-009):
//
//	mcp.classificationNudge        - structured nudge promoted from plain string in doc(action:"register")
//	mcp.classificationNudgeSection - section entry within classificationNudge.Outline
//	docint.Classification          - decoded via req.RequireString + json.Unmarshal in doc_intel_tool.go
//	docint.ConceptIntroEntry       - nested in Classification.ConceptsIntro, also JSON-decoded
func TestJSONStructTags(t *testing.T) {
	t.Parallel()

	types := []interface{}{
		classificationNudge{},
		classificationNudgeSection{},
		docint.Classification{},
		docint.ConceptIntroEntry{},
	}

	for _, v := range types {
		rt := reflect.TypeOf(v)
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			if tag := f.Tag.Get("json"); tag == "" {
				t.Errorf("struct %s field %s: missing json tag", rt.Name(), f.Name)
			}
		}
	}
}
