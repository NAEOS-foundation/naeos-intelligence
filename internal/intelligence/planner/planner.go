package planner

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/classifier"
	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
)

// Task is a single proposed execution step. Status is always proposed; plans
// are not authorization to execute.
type Task struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	DependsOn        []string `json:"depends_on,omitempty"`
	Priority         int      `json:"priority"` // 1=highest
	EstimatedMinutes int      `json:"estimated_minutes"`
	Status           string   `json:"status"`
}

// Plan is a structured, prioritized task plan with dependencies.
type Plan struct {
	RequestID  string            `json:"request_id"`
	Intent     classifier.Intent `json:"intent"`
	Tasks      []Task            `json:"tasks"`
	Summary    string            `json:"summary"`
	Provenance string            `json:"provenance"`
	CreatedAt  time.Time         `json:"created_at"`
}

// Planner generates a task plan from an intent and assembled context.
type Planner interface {
	Plan(ctx context.Context, requestID string, intent classifier.Intent, ctxx *intctx.Context, desiredOutputs []string) (Plan, error)
}

// RulePlanner generates deterministic task plans based on intent.
type RulePlanner struct{}

// NewRulePlanner creates a deterministic rule-based planner.
func NewRulePlanner() *RulePlanner {
	return &RulePlanner{}
}

func (p *RulePlanner) Plan(ctx context.Context, requestID string, intent classifier.Intent, ctxx *intctx.Context, desiredOutputs []string) (Plan, error) {
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	if strings.TrimSpace(requestID) == "" {
		return Plan{}, errors.New("request_id is required")
	}
	if ctxx == nil {
		return Plan{}, errors.New("context is required")
	}

	var tasks []Task
	var summary string
	switch intent {
	case classifier.IntentRepositoryAnalysis:
		tasks, summary = repositoryAnalysisTasks(requestID, ctxx)
	case classifier.IntentIssueInterpretation:
		tasks, summary = issueInterpretationTasks(requestID, ctxx)
	case classifier.IntentPRReview:
		tasks, summary = prReviewTasks(requestID, ctxx)
	case classifier.IntentPlanning:
		tasks, summary = planningTasks(requestID, ctxx, desiredOutputs)
	case classifier.IntentKnowledgeRetrieval:
		tasks, summary = knowledgeRetrievalTasks(requestID, ctxx)
	default:
		tasks, summary = unknownIntentTasks(requestID, ctxx)
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority < tasks[j].Priority
		}
		return tasks[i].ID < tasks[j].ID
	})

	return Plan{
		RequestID:  requestID,
		Intent:     intent,
		Tasks:      tasks,
		Summary:    summary,
		Provenance: "rule-based planning engine; tasks=" + itoa(len(tasks)),
		CreatedAt:  time.Now().UTC(),
	}, nil
}

func repositoryAnalysisTasks(requestID string, c *intctx.Context) ([]Task, string) {
	tasks := []Task{
		{ID: taskID(requestID, 1), Title: "Inventory repository structure", Description: "List directories, modules, and key entry points", DependsOn: nil, Priority: 1, EstimatedMinutes: 30, Status: "proposed"},
		{ID: taskID(requestID, 2), Title: "Assess technical debt", Description: "Identify high-complexity files and deprecated patterns", DependsOn: []string{taskID(requestID, 1)}, Priority: 2, EstimatedMinutes: 45, Status: "proposed"},
		{ID: taskID(requestID, 3), Title: "Document architecture risks", Description: "Summarize coupling, cohesion, and dependency risks", DependsOn: []string{taskID(requestID, 2)}, Priority: 3, EstimatedMinutes: 30, Status: "proposed"},
	}
	return tasks, "Proposed repository analysis: inventory, technical debt assessment, and architecture risk documentation"
}

func issueInterpretationTasks(requestID string, c *intctx.Context) ([]Task, string) {
	tasks := []Task{
		{ID: taskID(requestID, 1), Title: "Parse issue text", Description: "Extract problem statement, acceptance criteria, and constraints", DependsOn: nil, Priority: 1, EstimatedMinutes: 20, Status: "proposed"},
		{ID: taskID(requestID, 2), Title: "Map to codebase areas", Description: "Identify repositories, modules, and files relevant to the issue", DependsOn: []string{taskID(requestID, 1)}, Priority: 2, EstimatedMinutes: 30, Status: "proposed"},
		{ID: taskID(requestID, 3), Title: "Propose resolution approach", Description: "Draft implementation steps and verification criteria", DependsOn: []string{taskID(requestID, 2)}, Priority: 3, EstimatedMinutes: 40, Status: "proposed"},
	}
	return tasks, "Proposed issue interpretation: parse, map to codebase, and propose resolution"
}

func prReviewTasks(requestID string, c *intctx.Context) ([]Task, string) {
	tasks := []Task{
		{ID: taskID(requestID, 1), Title: "Review changed files", Description: "List all files modified in the pull request", DependsOn: nil, Priority: 1, EstimatedMinutes: 25, Status: "proposed"},
		{ID: taskID(requestID, 2), Title: "Identify risk areas", Description: "Flag security, performance, and breaking-change risks", DependsOn: []string{taskID(requestID, 1)}, Priority: 2, EstimatedMinutes: 35, Status: "proposed"},
		{ID: taskID(requestID, 3), Title: "Summarize findings", Description: "Produce review summary with approved/blocked recommendations", DependsOn: []string{taskID(requestID, 2)}, Priority: 3, EstimatedMinutes: 20, Status: "proposed"},
	}
	return tasks, "Proposed PR review: review changes, identify risks, and summarize findings"
}

func planningTasks(requestID string, c *intctx.Context, desiredOutputs []string) ([]Task, string) {
	tasks := []Task{
		{ID: taskID(requestID, 1), Title: "Define objectives", Description: "Clarify goals and success criteria from desired outputs", DependsOn: nil, Priority: 1, EstimatedMinutes: 20, Status: "proposed"},
		{ID: taskID(requestID, 2), Title: "Decompose into tasks", Description: "Break objectives into discrete, ordered tasks", DependsOn: []string{taskID(requestID, 1)}, Priority: 2, EstimatedMinutes: 40, Status: "proposed"},
		{ID: taskID(requestID, 3), Title: "Sequence and estimate", Description: "Order by dependency and assign effort estimates", DependsOn: []string{taskID(requestID, 2)}, Priority: 3, EstimatedMinutes: 30, Status: "proposed"},
	}
	if len(desiredOutputs) > 0 {
		tasks = append(tasks, Task{
			ID:               taskID(requestID, 4),
			Title:            "Validate against desired outputs",
			Description:      "Confirm plan covers each requested output",
			DependsOn:        []string{taskID(requestID, 3)},
			Priority:         4,
			EstimatedMinutes: 15,
			Status:           "proposed",
		})
	}
	return tasks, "Proposed planning: define objectives, decompose, sequence, and validate against desired outputs"
}

func knowledgeRetrievalTasks(requestID string, c *intctx.Context) ([]Task, string) {
	tasks := []Task{
		{ID: taskID(requestID, 1), Title: "Identify relevant sources", Description: "Select documentation, code, and issue sources to query", DependsOn: nil, Priority: 1, EstimatedMinutes: 15, Status: "proposed"},
		{ID: taskID(requestID, 2), Title: "Retrieve and summarize", Description: "Fetch source content and produce a concise summary", DependsOn: []string{taskID(requestID, 1)}, Priority: 2, EstimatedMinutes: 30, Status: "proposed"},
		{ID: taskID(requestID, 3), Title: "Cite provenance", Description: "Attach source references and confidence to each finding", DependsOn: []string{taskID(requestID, 2)}, Priority: 3, EstimatedMinutes: 15, Status: "proposed"},
	}
	return tasks, "Proposed knowledge retrieval: identify sources, retrieve, and cite provenance"
}

func unknownIntentTasks(requestID string, c *intctx.Context) ([]Task, string) {
	tasks := []Task{
		{ID: taskID(requestID, 1), Title: "Clarify intent", Description: "Request additional context to determine the appropriate intelligence task", DependsOn: nil, Priority: 1, EstimatedMinutes: 10, Status: "proposed"},
	}
	return tasks, "Proposed clarification: intent unknown, request additional context"
}

func taskID(requestID string, n int) string {
	return fmt.Sprintf("%s:t%d", requestID, n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b strings.Builder
	for n > 0 {
		b.WriteByte(byte('0' + n%10))
		n /= 10
	}
	runes := []rune(b.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
