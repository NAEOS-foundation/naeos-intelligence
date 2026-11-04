package httpapi

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestOpenAPISpecPassesOpenAPI31Validation runs the spec through a standards
// validator in addition to the repository's focused contract tests. External
// references remain disabled so validation cannot fetch arbitrary resources.
func TestOpenAPISpecPassesOpenAPI31Validation(t *testing.T) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	document, err := loader.LoadFromFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI 3.1 document: %v", err)
	}
}
