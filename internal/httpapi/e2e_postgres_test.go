package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/classifier"
	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/orchestrator"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/planner"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/postgres"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/ratelimit"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/runtime"
)

type e2eFakeVerifier struct {
	principal httpapi.Principal
	err       error
}

func (f e2eFakeVerifier) Verify(context.Context, string) (httpapi.Principal, error) {
	return f.principal, f.err
}

func e2ePostgresDSN(t *testing.T) string {
	t.Helper()
	if os.Getenv("NAEOS_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set NAEOS_POSTGRES_INTEGRATION=1 to run PostgreSQL E2E test")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL must be set for PostgreSQL E2E test")
	}
	return dsn
}

func e2ePostgresStore(t *testing.T, ctx context.Context) *postgres.Store {
	t.Helper()
	dsn := e2ePostgresDSN(t)
	if err := postgres.Up(ctx, dsn, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	pool, err := postgres.OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return postgres.New(pool)
}

func e2ePostgresHandler(t *testing.T, ctx context.Context) (http.Handler, *postgres.Store) {
	t.Helper()
	store := e2ePostgresStore(t, ctx)
	auth := httpapi.Authenticator{Verifier: e2eFakeVerifier{principal: httpapi.Principal{
		ClientID: "e2e-client",
		Scopes:   map[string]bool{"intelligence:write": true, "intelligence:read": true},
	}}}
	return httpapi.NewHandler(store, auth, ratelimit.NewLimiter(100, 100), httpapi.MustTestValidator(), nil), store
}

func e2ePostgresHandlerWithOrchestrator(t *testing.T, ctx context.Context) (http.Handler, *postgres.Store) {
	t.Helper()
	store := e2ePostgresStore(t, ctx)
	auth := httpapi.Authenticator{Verifier: e2eFakeVerifier{principal: httpapi.Principal{
		ClientID: "e2e-client",
		Scopes:   map[string]bool{"intelligence:write": true, "intelligence:read": true},
	}}}
	cl := classifier.NewRuleClassifier()
	cb := intctx.NewRuleBuilder()
	pl := planner.NewRulePlanner()
	reg := model.NewRegistry()
	_ = reg.Register(&mockProvider{id: "e2e-provider"})
	rp := runtime.NewProcessor(reg)
	orch := orchestrator.NewOrchestrator(cl, cb, pl, rp)
	return httpapi.NewHandler(store, auth, ratelimit.NewLimiter(100, 100), httpapi.MustTestValidator(), orch), store
}

func TestE2EPostgresSubmitGetCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h, store := e2ePostgresHandler(t, ctx)

	body := `{"purpose":"e2e submit get cancel","execution_mode":"analysis_only","desired_outputs":["trace"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer fake")
	req.Header.Set("Idempotency-Key", "e2e-idem-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d, want 202: %s", rec.Code, rec.Body.String())
	}
	var accepted httpapi.Accepted
	if err := json.Unmarshal(rec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode accepted: %v", err)
	}
	if accepted.Status != httpapi.StatusAccepted || accepted.ApprovalRequired {
		t.Fatalf("unexpected accepted: %+v", accepted)
	}

	// Get the request
	getReq := httptest.NewRequest(http.MethodGet, accepted.StatusURL, nil)
	getReq.Header.Set("Authorization", "Bearer fake")
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status=%d, want 200: %s", getRec.Code, getRec.Body.String())
	}
	var got httpapi.Request
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if got.RequestID != accepted.RequestID {
		t.Fatalf("request_id=%q, want %q", got.RequestID, accepted.RequestID)
	}

	// Cancel the request
	cancelReq := httptest.NewRequest(http.MethodPost, accepted.StatusURL+"/cancel", nil)
	cancelReq.Header.Set("Authorization", "Bearer fake")
	cancelRec := httptest.NewRecorder()
	h.ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusAccepted {
		t.Fatalf("cancel status=%d, want 202: %s", cancelRec.Code, cancelRec.Body.String())
	}
	var cancelled httpapi.CancellationAccepted
	if err := json.Unmarshal(cancelRec.Body.Bytes(), &cancelled); err != nil {
		t.Fatalf("decode cancelled: %v", err)
	}
	if cancelled.Status != httpapi.StatusCancelled {
		t.Fatalf("cancelled status=%q, want cancelled", cancelled.Status)
	}

	// Verify via store directly
	stored, err := store.Get(ctx, "e2e-client", accepted.RequestID)
	if err != nil || stored.Status != httpapi.StatusCancelled {
		t.Fatalf("stored status=%s err=%v", stored.Status, err)
	}
}

func TestE2EPostgresHealthAndReady(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h, _ := e2ePostgresHandler(t, ctx)

	// Health should return 200 (store is up)
	healthReq := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	healthRec := httptest.NewRecorder()
	h.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health status=%d, want 200: %s", healthRec.Code, healthRec.Body.String())
	}

	// Ready should return 200
	readyReq := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	readyRec := httptest.NewRecorder()
	h.ServeHTTP(readyRec, readyReq)
	if readyRec.Code != http.StatusOK {
		t.Fatalf("ready status=%d, want 200: %s", readyRec.Code, readyRec.Body.String())
	}
}

func TestE2EPostgresCrossClientIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store := e2ePostgresStore(t, ctx)

	authA := httpapi.Authenticator{Verifier: e2eFakeVerifier{principal: httpapi.Principal{
		ClientID: "client-a",
		Scopes:   map[string]bool{"intelligence:write": true, "intelligence:read": true},
	}}}
	authB := httpapi.Authenticator{Verifier: e2eFakeVerifier{principal: httpapi.Principal{
		ClientID: "client-b",
		Scopes:   map[string]bool{"intelligence:write": true, "intelligence:read": true},
	}}}
	hA := httpapi.NewHandler(store, authA, ratelimit.NewLimiter(100, 100), httpapi.MustTestValidator(), nil)
	hB := httpapi.NewHandler(store, authB, ratelimit.NewLimiter(100, 100), httpapi.MustTestValidator(), nil)

	// Submit as client-a
	body := `{"purpose":"cross client isolation","execution_mode":"analysis_only"}`
	submitReq := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(body))
	submitReq.Header.Set("Authorization", "Bearer fake")
	submitReq.Header.Set("Idempotency-Key", "iso-key-1")
	submitRec := httptest.NewRecorder()
	hA.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d, want 202: %s", submitRec.Code, submitRec.Body.String())
	}
	var accepted httpapi.Accepted
	if err := json.Unmarshal(submitRec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode accepted: %v", err)
	}

	// Client-b tries to get client-a's request -> 404
	getReq := httptest.NewRequest(http.MethodGet, accepted.StatusURL, nil)
	getReq.Header.Set("Authorization", "Bearer fake")
	getRec := httptest.NewRecorder()
	hB.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("cross-client get status=%d, want 404: %s", getRec.Code, getRec.Body.String())
	}

	// Client-a can still get its own request
	getRecA := httptest.NewRecorder()
	hA.ServeHTTP(getRecA, getReq)
	if getRecA.Code != http.StatusOK {
		t.Fatalf("owner get status=%d, want 200: %s", getRecA.Code, getRecA.Body.String())
	}
}

func TestE2EPostgresAnalyze(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h, _ := e2ePostgresHandlerWithOrchestrator(t, ctx)

	// Submit a request
	body := `{"purpose":"Analyze repository architecture and technical debt","execution_mode":"analysis_only","context_sources":[{"type":"repository","reference":"github.com/owner/repo"}],"desired_outputs":["risk_summary"]}`
	submitReq := httptest.NewRequest(http.MethodPost, "/api/v1/intelligence/requests", strings.NewReader(body))
	submitReq.Header.Set("Authorization", "Bearer fake")
	submitReq.Header.Set("Idempotency-Key", "e2e-analyze-idem-1")
	submitRec := httptest.NewRecorder()
	h.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusAccepted {
		t.Fatalf("submit status=%d, want 202: %s", submitRec.Code, submitRec.Body.String())
	}
	var accepted httpapi.Accepted
	if err := json.Unmarshal(submitRec.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode accepted: %v", err)
	}
	if accepted.RequestID == "" {
		t.Fatal("expected request_id in accepted response")
	}

	// Run analyze
	analyzeReq := httptest.NewRequest(http.MethodPost, accepted.StatusURL+"/analyze", strings.NewReader(`{}`))
	analyzeReq.Header.Set("Authorization", "Bearer fake")
	analyzeRec := httptest.NewRecorder()
	h.ServeHTTP(analyzeRec, analyzeReq)
	if analyzeRec.Code != http.StatusOK {
		t.Fatalf("analyze status=%d, want 200: %s", analyzeRec.Code, analyzeRec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(analyzeRec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode analyze result: %v", err)
	}
	if result["intent"] != "repository_analysis" {
		t.Errorf("intent=%v, want repository_analysis", result["intent"])
	}
	if result["request_id"] != accepted.RequestID {
		t.Errorf("request_id=%v, want %s", result["request_id"], accepted.RequestID)
	}
	plan, ok := result["plan"].(map[string]any)
	if !ok {
		t.Fatalf("plan not found in response: %v", result["plan"])
	}
	tasks, ok := plan["tasks"].([]any)
	if !ok || len(tasks) == 0 {
		t.Fatalf("plan.tasks missing or empty: %v", plan["tasks"])
	}
	runtimeResult, ok := result["runtime_result"].(map[string]any)
	if !ok {
		t.Fatalf("runtime_result not found in response: %v", result["runtime_result"])
	}
	if runtimeResult["allowed"] != true {
		t.Errorf("runtime_result.allowed=%v, want true", runtimeResult["allowed"])
	}
	policyDecision, ok := result["policy_decision"].(map[string]any)
	if !ok {
		t.Fatalf("policy_decision not found in response: %v", result["policy_decision"])
	}
	if policyDecision["allowed"] != true {
		t.Errorf("policy_decision.allowed=%v, want true", policyDecision["allowed"])
	}
	if _, ok := result["provenance"].(string); !ok {
		t.Errorf("provenance missing or not a string: %v", result["provenance"])
	}
}

type mockProvider struct {
	id string
}

func (m *mockProvider) ID() string { return m.id }
func (m *mockProvider) Generate(context.Context, model.Request) (model.Response, error) {
	return model.Response{ModelID: m.id, Text: "fake"}, nil
}
func (m *mockProvider) HealthCheck(context.Context) error { return nil }
func (m *mockProvider) Capabilities() []string            { return []string{"text"} }
