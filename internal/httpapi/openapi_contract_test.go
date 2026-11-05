package httpapi

import (
	"os"
	"strings"
	"testing"
)

func readOpenAPISpec(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI spec: %v", err)
	}
	return string(data)
}

func sectionBetween(t *testing.T, source, start, end string) string {
	t.Helper()
	startAt := strings.Index(source, start)
	if startAt < 0 {
		t.Fatalf("OpenAPI section start not found: %q", start)
	}
	startAt += len(start)
	if end == "" {
		return source[startAt:]
	}
	endAt := strings.Index(source[startAt:], end)
	if endAt < 0 {
		t.Fatalf("OpenAPI section end not found: %q", end)
	}
	return source[startAt : startAt+endAt]
}

func assertOpenAPIContains(t *testing.T, section string, required ...string) {
	t.Helper()
	for _, item := range required {
		if !strings.Contains(section, item) {
			t.Errorf("OpenAPI section missing contract fragment %q", item)
		}
	}
}

func TestOpenAPIOperationContracts(t *testing.T) {
	spec := readOpenAPISpec(t)
	submit := sectionBetween(t, spec, "  /intelligence/requests:\n", "  /intelligence/requests/{request_id}:\n")
	assertOpenAPIContains(t, submit,
		"operationId: submitIntelligenceRequest",
		"\"202\":",
		"#/components/schemas/Accepted",
		"\"400\":\n          $ref: \"#/components/responses/Problem\"",
		"\"401\":\n          $ref: \"#/components/responses/Problem\"",
		"\"403\":\n          $ref: \"#/components/responses/Problem\"",
		"\"409\":\n          $ref: \"#/components/responses/Problem\"",
		"\"422\":\n          $ref: \"#/components/responses/Problem\"",
		"\"503\":\n          $ref: \"#/components/responses/Problem\"",
	)

	getRequest := sectionBetween(t, spec, "  /intelligence/requests/{request_id}:\n", "  /intelligence/requests/{request_id}/cancel:\n")
	assertOpenAPIContains(t, getRequest,
		"operationId: getIntelligenceRequest",
		"\"200\":",
		"#/components/schemas/Request",
		"\"404\":\n          $ref: \"#/components/responses/Problem\"",
	)

	cancel := sectionBetween(t, spec, "  /intelligence/requests/{request_id}/cancel:\n", "components:\n")
	assertOpenAPIContains(t, cancel,
		"operationId: cancelIntelligenceRequest",
		"\"202\":",
		"#/components/schemas/CancellationAccepted",
		"\"404\":\n          $ref: \"#/components/responses/Problem\"",
		"\"409\":\n          $ref: \"#/components/responses/Problem\"",
	)

	ready := sectionBetween(t, spec, "  /ready:\n", "  /intelligence/requests:\n")
	assertOpenAPIContains(t, ready,
		"operationId: getReady",
		"\"200\":",
		"\"503\":\n          $ref: \"#/components/responses/Problem\"",
	)
}

func TestOpenAPIProblemEnvelopeAndInputBounds(t *testing.T) {
	spec := readOpenAPISpec(t)
	problemResponse := sectionBetween(t, spec, "  responses:\n    Problem:\n", "  schemas:\n")
	assertOpenAPIContains(t, problemResponse,
		"application/problem+json:",
		"#/components/schemas/Problem",
		"X-Request-ID:",
	)

	problemSchema := sectionBetween(t, spec, "    Problem:\n", "")
	assertOpenAPIContains(t, problemSchema,
		"required: [type, title, status, code, detail, request_id]",
		"type: { type: string, format: uri }",
		"title: { type: string }",
		"status: { type: integer }",
		"code: { type: string }",
		"detail: { type: string }",
		"request_id: { type: string }",
	)

	requestInput := sectionBetween(t, spec, "    RequestInput:\n", "    Accepted:\n")
	assertOpenAPIContains(t, requestInput,
		"purpose: { type: string, minLength: 1, maxLength: 2000 }",
		"maxItems: 100",
		"execution_mode: { type: string, const: analysis_only }",
		"maxItems: 50",
		"maxProperties: 50",
		"additionalProperties: { type: string, maxLength: 256 }",
	)

	requestSchema := sectionBetween(t, spec, "    Request:\n", "    CancellationAccepted:\n")
	assertOpenAPIContains(t, requestSchema,
		"approval_required: { type: boolean }",
		"execution_mode: { type: string, const: analysis_only }",
	)
}
