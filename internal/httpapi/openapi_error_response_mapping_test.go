package httpapi

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestOpenAPIErrorResponsesDeclareEndpointCodes ensures each documented
// endpoint-level error status identifies its stable public error code while
// retaining the shared Problem envelope.
func TestOpenAPIErrorResponsesDeclareEndpointCodes(t *testing.T) {
	data, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI spec: %v", err)
	}

	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse OpenAPI YAML: %v", err)
	}
	paths := mapping(t, document["paths"], "paths")
	operations := []struct {
		path, method, operationID string
		errors                    map[string]string
	}{
		{
			"/intelligence/requests", "post", "submitIntelligenceRequest",
			map[string]string{
				"400": "INVALID_REQUEST", "401": "UNAUTHENTICATED",
				"403": "FORBIDDEN", "409": "CONFLICT",
				"413": "PAYLOAD_TOO_LARGE", "422": "UNPROCESSABLE_REQUEST",
				"429": "RATE_LIMITED", "500": "INTERNAL_ERROR",
				"503": "SERVICE_UNAVAILABLE",
			},
		},
		{
			"/intelligence/requests/{request_id}", "get", "getIntelligenceRequest",
			map[string]string{
				"401": "UNAUTHENTICATED", "403": "FORBIDDEN",
				"404": "RESOURCE_NOT_FOUND", "429": "RATE_LIMITED",
				"500": "INTERNAL_ERROR",
				"503": "SERVICE_UNAVAILABLE",
			},
		},
		{
			"/intelligence/requests/{request_id}/cancel", "post", "cancelIntelligenceRequest",
			map[string]string{
				"401": "UNAUTHENTICATED", "403": "FORBIDDEN",
				"404": "RESOURCE_NOT_FOUND", "409": "CONFLICT",
				"429": "RATE_LIMITED", "500": "INTERNAL_ERROR",
				"503": "SERVICE_UNAVAILABLE",
			},
		},
		{
			"/intelligence/requests/{request_id}/analyze", "post", "analyzeIntelligenceRequest",
			map[string]string{
				"400": "INVALID_REQUEST", "401": "UNAUTHENTICATED",
				"403": "FORBIDDEN", "404": "RESOURCE_NOT_FOUND",
				"413": "PAYLOAD_TOO_LARGE", "422": "UNPROCESSABLE_REQUEST",
				"429": "RATE_LIMITED", "500": "INTERNAL_ERROR",
				"503": "SERVICE_UNAVAILABLE",
			},
		},
	}

	for _, expected := range operations {
		pathItem := mapping(t, paths[expected.path], "paths."+expected.path)
		op := mapping(t, pathItem[expected.method], expected.operationID)
		responses := mapping(t, op["responses"], expected.operationID+".responses")
		for status, code := range expected.errors {
			response := mapping(t, responses[status], expected.operationID+".responses."+status)
			description, ok := response["description"].(string)
			if !ok || !strings.Contains(description, code) {
				t.Errorf("%s response %s description must identify %s; got %q", expected.operationID, status, code, description)
			}
			ref, ok := response["$ref"].(string)
			if !ok || ref != "#/components/responses/Problem" {
				t.Errorf("%s response %s must reference the shared Problem envelope; got %v", expected.operationID, status, response["$ref"])
			}
		}
	}
}
