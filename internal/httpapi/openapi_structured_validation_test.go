package httpapi

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestOpenAPISpecIsStructuredAndReferencesResolve supplements the source-fragment
// checks with YAML parsing, operation-level contract checks, and local $ref
// resolution. It intentionally does not claim full OpenAPI schema validation.
func TestOpenAPISpecIsStructuredAndReferencesResolve(t *testing.T) {
	data, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI spec: %v", err)
	}

	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse OpenAPI YAML: %v", err)
	}
	if document["openapi"] != "3.1.0" {
		t.Fatalf("OpenAPI version = %v, want 3.1.0", document["openapi"])
	}
	info := mapping(t, document["info"], "info")
	if info["title"] != "NAEOS Intelligence API" || info["version"] != "0.1.0" {
		t.Fatalf("unexpected API info: %#v", info)
	}
	servers := sequence(t, document["servers"], "servers")
	if len(servers) == 0 || mapping(t, servers[0], "servers[0]")["url"] != "/api/v1" {
		t.Fatal("OpenAPI server URL must be /api/v1")
	}

	paths := mapping(t, document["paths"], "paths")
	operations := []struct {
		path, method, operationID, scope string
		responses                        []string
	}{
		{"/health", "get", "getHealth", "", []string{"200"}},
		{"/intelligence/requests", "post", "submitIntelligenceRequest", "intelligence:write", []string{"202", "400", "401", "403", "409", "413", "422", "503"}},
		{"/intelligence/requests/{request_id}", "get", "getIntelligenceRequest", "intelligence:read", []string{"200", "401", "403", "404", "503"}},
		{"/intelligence/requests/{request_id}/cancel", "post", "cancelIntelligenceRequest", "intelligence:write", []string{"202", "401", "403", "404", "409", "503"}},
	}
	for _, expected := range operations {
		pathItem := mapping(t, paths[expected.path], "paths."+expected.path)
		op := mapping(t, pathItem[expected.method], expected.path+"."+expected.method)
		if op["operationId"] != expected.operationID {
			t.Errorf("%s %s operationId = %v, want %s", expected.method, expected.path, op["operationId"], expected.operationID)
		}
		if expected.scope == "" {
			security, exists := op["security"]
			if !exists || len(sequence(t, security, "security")) != 0 {
				t.Errorf("%s must explicitly disable global security", expected.operationID)
			}
		} else {
			security := sequence(t, op["security"], expected.operationID+".security")
			if len(security) != 1 {
				t.Errorf("%s must declare one security requirement", expected.operationID)
			} else {
				requirement := mapping(t, security[0], expected.operationID+".security[0]")
				scopes := sequence(t, requirement["bearerAuth"], expected.operationID+".bearerAuth")
				if len(scopes) != 1 || scopes[0] != expected.scope {
					t.Errorf("%s bearerAuth scopes = %v, want [%s]", expected.operationID, scopes, expected.scope)
				}
			}
		}
		responses := mapping(t, op["responses"], expected.operationID+".responses")
		for _, code := range expected.responses {
			if _, exists := responses[code]; !exists {
				t.Errorf("%s missing response %s", expected.operationID, code)
			}
		}
	}

	components := mapping(t, document["components"], "components")
	schemas := mapping(t, components["schemas"], "components.schemas")
	for _, name := range []string{"RequestInput", "Accepted", "Request", "CancellationAccepted", "Status", "Problem"} {
		if _, exists := schemas[name]; !exists {
			t.Errorf("components.schemas missing %q", name)
		}
	}
	assertLocalRefsResolve(t, document, document)
}

func mapping(t *testing.T, value any, path string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s must be a mapping, got %T", path, value)
	}
	return result
}

func sequence(t *testing.T, value any, path string) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("%s must be a sequence, got %T", path, value)
	}
	return result
}

func assertLocalRefsResolve(t *testing.T, value any, root map[string]any) {
	t.Helper()
	switch node := value.(type) {
	case map[string]any:
		if ref, ok := node["$ref"].(string); ok && strings.HasPrefix(ref, "#/") {
			current := any(root)
			for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
				m, ok := current.(map[string]any)
				if !ok {
					t.Errorf("$ref %q traverses non-mapping at %q", ref, part)
					break
				}
				next, exists := m[part]
				if !exists {
					t.Errorf("unresolved local $ref %q (missing %q)", ref, part)
					break
				}
				current = next
			}
		}
		for _, child := range node {
			assertLocalRefsResolve(t, child, root)
		}
	case []any:
		for _, child := range node {
			assertLocalRefsResolve(t, child, root)
		}
	}
}
