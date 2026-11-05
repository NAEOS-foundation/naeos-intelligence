package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// OpenAPIValidator validates incoming HTTP requests against the OpenAPI 3.1 spec.
type OpenAPIValidator struct {
	router routers.Router
}

const maxBodyBytes = 1 << 20

// NewOpenAPIValidator loads the OpenAPI spec and creates a request validator.
func NewOpenAPIValidator(specPath string) (*OpenAPIValidator, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec: %w", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate OpenAPI spec: %w", err)
	}
	router, err := legacy.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("create OpenAPI router: %w", err)
	}
	return &OpenAPIValidator{router: router}, nil
}

// Middleware returns an HTTP middleware that validates requests against the OpenAPI spec.
func (v *OpenAPIValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, pathParams, err := v.router.FindRoute(r)
		if err != nil {
			// Route not in spec; let the handler deal with it (e.g. 404).
			next.ServeHTTP(w, r)
			return
		}

		// Pre-validate body size and JSON decodability before schema validation,
		// so that oversized or malformed bodies get the correct HTTP status.
		if validateBodySizeAndJSON(w, r) {
			return
		}

		input := &openapi3filter.RequestValidationInput{
			Request:    r,
			Route:      route,
			PathParams: pathParams,
			QueryParams: r.URL.Query(),
			Options: &openapi3filter.Options{
				ExcludeRequestBody:  false,
				AuthenticationFunc:  openapi3filter.NoopAuthenticationFunc,
			},
		}

		if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
			writeProblem(w, r, http.StatusUnprocessableEntity, "UNPROCESSABLE_REQUEST",
				fmt.Sprintf("request validation failed: %v", err))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validateBodySizeAndJSON checks that the request body is within the size limit
// and is valid JSON (when a body is present). Returns true if the response was
// already written (caller should stop processing).
func validateBodySizeAndJSON(w http.ResponseWriter, r *http.Request) bool {
	if r.Body == nil {
		return false
	}

	// Enforce body size limit.
	limited := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	data, err := io.ReadAll(limited)
	if err != nil {
		if strings.Contains(err.Error(), "request body exceeds") {
			writeProblem(w, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
				"request body exceeds 1 MiB")
		} else {
			writeProblem(w, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE",
				"failed to read request body")
		}
		return true
	}

	// Restore body for downstream handlers.
	r.Body = io.NopCloser(strings.NewReader(string(data)))

	// If there's a body, check it's valid JSON.
	if len(data) > 0 {
		var js any
		if err := json.Unmarshal(data, &js); err != nil {
			writeProblem(w, r, http.StatusBadRequest, "INVALID_REQUEST",
				"request body must be valid JSON")
			return true
		}
	}

	return false
}
