package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// OpenAPIValidator validates incoming HTTP requests and outgoing responses
// against the OpenAPI 3.1 spec.
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

// Middleware returns an HTTP middleware that validates requests and responses
// against the OpenAPI spec.
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

		// Wrap the response writer to capture status and body for validation.
		capture := &responseCaptureWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(capture, r)

		// Validate the response body against the OpenAPI schema.
		// Validation failures are logged but do not alter the client response,
		// to avoid breaking clients due to server-side validation issues.
		if err := v.validateResponse(route, capture.status, capture.body.Bytes()); err != nil {
			log.Printf("OpenAPI response validation failed: %v", err)
		}
	})
}

// responseCaptureWriter wraps http.ResponseWriter to capture status code and body.
type responseCaptureWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *responseCaptureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseCaptureWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
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

// validateResponse validates the response body against the OpenAPI schema
// for the given operation and status code.
func (v *OpenAPIValidator) validateResponse(route *routers.Route, statusCode int, body []byte) error {
	if route == nil || route.Operation == nil || route.Operation.Responses == nil {
		return nil
	}

	// Skip validation for empty bodies (e.g. 204 No Content).
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}

	resp := route.Operation.Responses.Status(statusCode)
	if resp == nil || resp.Value == nil || resp.Value.Content == nil {
		resp = route.Operation.Responses.Default()
	}
	if resp == nil || resp.Value == nil || resp.Value.Content == nil {
		return nil
	}

	mediaType := resp.Value.Content["application/json"]
	if mediaType == nil || mediaType.Schema == nil || mediaType.Schema.Value == nil {
		return nil
	}

	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	err := mediaType.Schema.Value.VisitJSON(value, openapi3.EnableJSONSchema2020())
	if err != nil {
		return fmt.Errorf("response body violates schema: %w", err)
	}
	return nil
}
