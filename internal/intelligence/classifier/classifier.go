package classifier

import (
	"context"
	"errors"
	"strings"
)

// Intent is a classified engineering intelligence intent.
type Intent string

const (
	IntentUnknown             Intent = "unknown"
	IntentRepositoryAnalysis  Intent = "repository_analysis"
	IntentIssueInterpretation Intent = "issue_interpretation"
	IntentPRReview            Intent = "pr_review"
	IntentPlanning            Intent = "planning"
	IntentKnowledgeRetrieval  Intent = "knowledge_retrieval"
)

// Result is the deterministic classification output. It is a proposal, not
// authorization, and carries no side effects.
type Result struct {
	Intent     Intent
	Confidence float64
	Reason     string
	Evidence   []string
}

// Classifier classifies a request purpose into an engineering intent.
type Classifier interface {
	Classify(ctx context.Context, purpose string, contextSources []ContextSource) (Result, error)
}

// ContextSource describes an optional source of engineering context.
type ContextSource struct {
	Type      string
	Reference string
}

// RuleClassifier implements Classifier using deterministic keyword and source
// type heuristics. No model invocation is performed.
type RuleClassifier struct{}

// NewRuleClassifier creates a deterministic rule-based classifier.
func NewRuleClassifier() *RuleClassifier {
	return &RuleClassifier{}
}

// Classify classifies purpose and context sources into an intent.
func (c *RuleClassifier) Classify(ctx context.Context, purpose string, contextSources []ContextSource) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trimmed := strings.TrimSpace(purpose)
	if trimmed == "" {
		return Result{}, errors.New("purpose is required")
	}
	lower := strings.ToLower(trimmed)
	sourceTypes := sourceTypeSet(contextSources)

	intent, confidence, reason, evidence := classify(lower, sourceTypes)
	return Result{
		Intent:     intent,
		Confidence: confidence,
		Reason:     reason,
		Evidence:   evidence,
	}, nil
}

func classify(lower string, sourceTypes map[string]struct{}) (Intent, float64, string, []string) {
	evidence := make([]string, 0, len(sourceTypes))
	for t := range sourceTypes {
		evidence = append(evidence, "context_source:"+t)
	}

	keywords := map[Intent][]string{
		IntentRepositoryAnalysis:  {"repository", "repo", "codebase", "structure", "architecture", "technical debt"},
		IntentIssueInterpretation: {"issue", "ticket", "requirement", "bug", "feature request", "interpret"},
		IntentPRReview:            {"pull request", "pr ", "merge request", "code review", "diff", "changeset"},
		IntentPlanning:            {"plan", "planning", "task", "schedule", "roadmap", "dependency", "execution plan"},
		IntentKnowledgeRetrieval:  {"retrieve", "knowledge", "search", "documentation", "context", "lookup"},
	}

	bestIntent := IntentUnknown
	bestConfidence := 0.0
	bestReason := "no matching intent keywords; classified as unknown"

	for intent, terms := range keywords {
		hits := 0
		for _, term := range terms {
			if strings.Contains(lower, term) {
				hits++
			}
		}
		if hits == 0 {
			continue
		}
		confidence := intentConfidence(hits, len(terms), len(sourceTypes))
		if confidence > bestConfidence {
			bestIntent = intent
			bestConfidence = confidence
			bestReason = intentReason(intent, hits)
		}
	}

	if bestIntent == IntentUnknown && len(sourceTypes) > 0 {
		bestIntent = IntentKnowledgeRetrieval
		bestConfidence = 0.4
		bestReason = "classified from context sources without matching purpose keywords"
	}

	return bestIntent, bestConfidence, bestReason, evidence
}

func intentConfidence(hits, totalTerms int, sourceCount int) float64 {
	base := float64(hits) / float64(totalTerms)
	if base > 1.0 {
		base = 1.0
	}
	if sourceCount > 0 {
		base += 0.1
		if base > 0.95 {
			base = 0.95
		}
	}
	return base
}

func intentReason(intent Intent, hits int) string {
	switch intent {
	case IntentRepositoryAnalysis:
		return "purpose matches repository analysis keywords"
	case IntentIssueInterpretation:
		return "purpose matches issue interpretation keywords"
	case IntentPRReview:
		return "purpose matches PR review keywords"
	case IntentPlanning:
		return "purpose matches planning keywords"
	case IntentKnowledgeRetrieval:
		return "purpose matches knowledge retrieval keywords"
	default:
		return "no matching intent"
	}
}

func sourceTypeSet(sources []ContextSource) map[string]struct{} {
	set := make(map[string]struct{}, len(sources))
	for _, s := range sources {
		if t := strings.TrimSpace(s.Type); t != "" {
			set[t] = struct{}{}
		}
	}
	return set
}
