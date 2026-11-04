package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	policy "github.com/NAEOS-foundation/naeos-intelligence/internal/policy"
)

const maxBodyBytes = 1 << 20

var correlationPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var errPayloadTooLarge = errors.New("request body exceeds 1 MiB")

type Handler struct {
	store Store
	auth  Authenticator
	mux   *http.ServeMux
}

func NewHandler(store Store, auth Authenticator) http.Handler {
	h := &Handler{store: store, auth: auth, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /api/v1/health", h.health)
	h.mux.Handle("POST /api/v1/intelligence/requests", auth.Middleware("intelligence:write", http.HandlerFunc(h.submit)))
	h.mux.Handle("GET /api/v1/intelligence/requests/{request_id}", auth.Middleware("intelligence:read", http.HandlerFunc(h.get)))
	h.mux.Handle("POST /api/v1/intelligence/requests/{request_id}/cancel", auth.Middleware("intelligence:write", http.HandlerFunc(h.cancel)))
	return h.requestID(h.mux)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFrom(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
		return
	}
	var input RequestInput
	if err := decodeBody(w, r, &input); err != nil {
		status := http.StatusBadRequest
		code := "INVALID_REQUEST"
		if errors.Is(err, errPayloadTooLarge) {
			status = http.StatusRequestEntityTooLarge
			code = "PAYLOAD_TOO_LARGE"
		}
		writeProblem(w, r, status, code, err.Error())
		return
	}
	if err := validateInput(input); err != nil {
		writeProblem(w, r, http.StatusUnprocessableEntity, "UNPROCESSABLE_REQUEST", err.Error())
		return
	}
	input.Purpose = strings.TrimSpace(input.Purpose)
	for i := range input.ContextSources {
		input.ContextSources[i].Type = strings.TrimSpace(input.ContextSources[i].Type)
		input.ContextSources[i].Reference = strings.TrimSpace(input.ContextSources[i].Reference)
	}
	for i := range input.DesiredOutputs {
		input.DesiredOutputs[i] = strings.TrimSpace(input.DesiredOutputs[i])
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(key) > 255 {
		writeProblem(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Idempotency-Key must not exceed 255 characters")
		return
	}
	req, replay, err := h.store.Create(r.Context(), principal.ClientID, key, input, requestIDFrom(r))
	if errors.Is(err, ErrIdempotencyConflict) {
		writeProblem(w, r, http.StatusConflict, "CONFLICT", "Idempotency key was used with a different request payload")
		return
	}
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Request could not be accepted")
		return
	}
	statusURL := "/api/v1/intelligence/requests/" + req.RequestID
	w.Header().Set("Location", statusURL)
	if replay {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	writeJSON(w, http.StatusAccepted, Accepted{RequestID: req.RequestID, Status: req.Status, ApprovalRequired: req.ApprovalRequired, CreatedAt: req.CreatedAt, StatusURL: statusURL})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
		return
	}
	req, err := h.store.Get(r.Context(), p.ClientID, r.PathValue("request_id"))
	if errors.Is(err, ErrNotFound) {
		writeProblem(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "Request was not found")
		return
	}
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Request could not be retrieved")
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
		return
	}
	req, err := h.store.Cancel(r.Context(), p.ClientID, r.PathValue("request_id"), requestIDFrom(r))
	if errors.Is(err, ErrNotFound) {
		writeProblem(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "Request was not found")
		return
	}
	if errors.Is(err, ErrConflict) {
		writeProblem(w, r, http.StatusConflict, "CONFLICT", "Request cannot be cancelled in its current state")
		return
	}
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Cancellation could not be processed")
		return
	}
	writeJSON(w, http.StatusAccepted, CancellationAccepted{RequestID: req.RequestID, Status: req.Status, Message: "Cancellation was accepted or request is already cancelled"})
}

func validateInput(in RequestInput) error {
	if strings.TrimSpace(in.Purpose) == "" || len(in.Purpose) > 2000 {
		return errors.New("purpose must contain 1 to 2000 characters")
	}
	if _, err := policy.EvaluateRequest(policy.RequestInput{
		Purpose:       in.Purpose,
		ExecutionMode: in.ExecutionMode,
	}); err != nil {
		return err
	}
	if len(in.ContextSources) > 100 {
		return errors.New("context_sources must contain at most 100 items")
	}
	for _, source := range in.ContextSources {
		if strings.TrimSpace(source.Type) == "" || len(source.Type) > 64 || strings.TrimSpace(source.Reference) == "" || len(source.Reference) > 1024 {
			return errors.New("context_sources contain an invalid item")
		}
	}
	if len(in.DesiredOutputs) > 50 {
		return errors.New("desired_outputs must contain at most 50 items")
	}
	for _, output := range in.DesiredOutputs {
		if strings.TrimSpace(output) == "" || len(output) > 128 {
			return errors.New("desired_outputs contain an invalid item")
		}
	}
	for key, value := range in.Metadata {
		if strings.TrimSpace(key) == "" || len(key) > 128 || len(value) > 256 {
			return errors.New("metadata contains an invalid item")
		}
	}
	return nil
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return errPayloadTooLarge
		}
		return errors.New("request body must be valid JSON")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

func (h *Handler) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if !correlationPattern.MatchString(id) {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func newRequestID() string { return time.Now().UTC().Format("20060102T150405.000000000Z") }
