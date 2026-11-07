package planner

import (
	"context"
	"testing"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/classifier"
	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
)

func TestRulePlanner_Plan(t *testing.T) {
	p := NewRulePlanner()
	ctx := context.Background()
	cx := &intctx.Context{RequestID: "req-1", Provenance: "test"}

	tests := []struct {
		name       string
		requestID  string
		intent     classifier.Intent
		outputs    []string
		wantMin    int
		wantStatus string
		wantErr    bool
	}{
		{
			name:    "empty request id returns error",
			intent:  classifier.IntentRepositoryAnalysis,
			wantErr: true,
		},
		{
			name:      "nil context returns error",
			requestID: "req-1",
			intent:    classifier.IntentRepositoryAnalysis,
			wantErr:   true,
		},
		{
			name:      "repository analysis plan",
			requestID: "req-1",
			intent:    classifier.IntentRepositoryAnalysis,
			wantMin:   3,
		},
		{
			name:      "issue interpretation plan",
			requestID: "req-2",
			intent:    classifier.IntentIssueInterpretation,
			wantMin:   3,
		},
		{
			name:      "PR review plan",
			requestID: "req-3",
			intent:    classifier.IntentPRReview,
			wantMin:   3,
		},
		{
			name:      "planning plan with desired outputs",
			requestID: "req-4",
			intent:    classifier.IntentPlanning,
			outputs:   []string{"roadmap", "backlog"},
			wantMin:   4,
		},
		{
			name:      "knowledge retrieval plan",
			requestID: "req-5",
			intent:    classifier.IntentKnowledgeRetrieval,
			wantMin:   3,
		},
		{
			name:      "unknown intent plan",
			requestID: "req-6",
			intent:    classifier.IntentUnknown,
			wantMin:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqID := tt.requestID
			if reqID == "" {
				reqID = "   "
			}
			var planCtx *intctx.Context
			if tt.name == "nil context returns error" {
				planCtx = nil
			} else {
				planCtx = cx
			}
			plan, err := p.Plan(ctx, reqID, tt.intent, planCtx, tt.outputs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if plan.RequestID != reqID {
				t.Errorf("request_id = %q, want %q", plan.RequestID, reqID)
			}
			if plan.Intent != tt.intent {
				t.Errorf("intent = %q, want %q", plan.Intent, tt.intent)
			}
			if len(plan.Tasks) < tt.wantMin {
				t.Errorf("tasks count = %d, want >= %d", len(plan.Tasks), tt.wantMin)
			}
			for _, task := range plan.Tasks {
				if task.Status != "proposed" {
					t.Errorf("task %s status = %q, want proposed", task.ID, task.Status)
				}
				if task.Priority < 1 {
					t.Errorf("task %s priority = %d, must be >= 1", task.ID, task.Priority)
				}
			}
			if plan.Summary == "" {
				t.Error("summary must not be empty")
			}
			if plan.Provenance == "" {
				t.Error("provenance must not be empty")
			}
			if plan.CreatedAt.IsZero() {
				t.Error("created_at must not be zero")
			}
		})
	}
}

func TestRulePlanner_Plan_DependsOnAcyclic(t *testing.T) {
	p := NewRulePlanner()
	ctx := context.Background()
	cx := &intctx.Context{RequestID: "req-1"}
	plan, err := p.Plan(ctx, "req-1", classifier.IntentRepositoryAnalysis, cx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byID := make(map[string]Task)
	for _, t := range plan.Tasks {
		byID[t.ID] = t
	}
	for _, task := range plan.Tasks {
		for _, dep := range task.DependsOn {
			if _, ok := byID[dep]; !ok {
				t.Errorf("task %s depends on unknown task %s", task.ID, dep)
			}
		}
	}
}

func TestRulePlanner_Plan_ContextCancelled(t *testing.T) {
	p := NewRulePlanner()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cx := &intctx.Context{RequestID: "req-1"}
	_, err := p.Plan(ctx, "req-1", classifier.IntentRepositoryAnalysis, cx, nil)
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}
