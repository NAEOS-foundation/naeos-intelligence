package httpapi

import (
	"os"
	"strings"
	"testing"
)

// TestOpenAPIOperationsMatchHTTPRoutes keeps the documented operation paths,
// methods, operation IDs, and required scopes aligned with the registered mux.
func TestOpenAPIOperationsMatchHTTPRoutes(t *testing.T) {
	handlerBytes, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatalf("read HTTP handler: %v", err)
	}
	handlerSource := string(handlerBytes)
	spec := readOpenAPISpec(t)

	if !strings.Contains(spec, "servers:\n  - url: /api/v1") {
		t.Fatal("OpenAPI server prefix must match the API version prefix registered by the handler")
	}

	cases := []struct {
		name, route, path, method, operationID, scope string
	}{
		{
			name: "health", route: `h.mux.HandleFunc("GET /api/v1/health", h.health)`,
			path: "  /health:\n", method: "get:", operationID: "getHealth",
		},
		{
			name: "ready", route: `h.mux.HandleFunc("GET /api/v1/ready", h.ready)`,
			path: "  /ready:\n", method: "get:", operationID: "getReady",
		},
		{
			name: "submit", route: `h.mux.Handle("POST /api/v1/intelligence/requests", auth.Middleware("intelligence:write", http.HandlerFunc(h.submit)))`,
			path: "  /intelligence/requests:\n", method: "post:", operationID: "submitIntelligenceRequest", scope: "intelligence:write",
		},
		{
			name: "retrieve", route: `h.mux.Handle("GET /api/v1/intelligence/requests/{request_id}", auth.Middleware("intelligence:read", http.HandlerFunc(h.get)))`,
			path: "  /intelligence/requests/{request_id}:\n", method: "get:", operationID: "getIntelligenceRequest", scope: "intelligence:read",
		},
		{
			name: "cancel", route: `h.mux.Handle("POST /api/v1/intelligence/requests/{request_id}/cancel", auth.Middleware("intelligence:write", http.HandlerFunc(h.cancel)))`,
			path: "  /intelligence/requests/{request_id}/cancel:\n", method: "post:", operationID: "cancelIntelligenceRequest", scope: "intelligence:write",
		},
		{
			name: "approve", route: `h.mux.Handle("POST /api/v1/intelligence/requests/{request_id}/approval", auth.Middleware("intelligence:write", http.HandlerFunc(h.approve)))`,
			path: "  /intelligence/requests/{request_id}/approval:\n", method: "post:", operationID: "recordApprovalDecision", scope: "intelligence:write",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(handlerSource, tc.route) {
				t.Fatalf("HTTP handler route or middleware registration changed: %s", tc.route)
			}
			pathSection := sectionBetween(t, spec, tc.path, "")
			methodSection := sectionBetween(t, pathSection, "    "+tc.method+"\n", "")
			if !strings.Contains(methodSection, "operationId: "+tc.operationID) {
				t.Fatalf("OpenAPI operation ID missing or mismatched: %s", tc.operationID)
			}
			if tc.scope != "" && !strings.Contains(methodSection, "bearerAuth: ["+tc.scope+"]") {
				t.Fatalf("OpenAPI security scope %q does not match handler middleware", tc.scope)
			}
			if tc.scope == "" && !strings.Contains(methodSection, "security: []") {
				t.Fatal("public health operation must explicitly disable global security")
			}
		})
	}
}
