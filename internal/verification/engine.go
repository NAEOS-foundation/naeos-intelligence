// Package verification implements the verification engine that evaluates
// execution outcomes against expected results.
package verification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Result is the outcome of a verification.
type Result string

const (
	// ResultVerified means the observed outcome matches the expected outcome.
	ResultVerified Result = "verified"
	// ResultFailed means the observed outcome does not match the expected outcome.
	ResultFailed Result = "failed"
	// ResultInconclusive means there is insufficient evidence to determine the outcome.
	ResultInconclusive Result = "inconclusive"
)

// VerificationRecord is the output of a verification.
type VerificationRecord struct {
	VerificationID  string    `json:"verification_id"`
	ExecutionID     string    `json:"execution_id"`
	Method          string    `json:"method"`
	ExpectedOutcome string    `json:"expected_outcome"`
	ObservedOutcome string    `json:"observed_outcome"`
	Result          Result    `json:"result"`
	VerifiedAt      time.Time `json:"verified_at"`
}

// Engine verifies execution outcomes against expected results.
type Engine struct{}

// New creates a verification engine.
func New() *Engine { return &Engine{} }

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:], nil
}

// Verify compares the observed outcome with the expected outcome and produces
// a verification record. The result is verified when they match, failed when
// they differ and observed is non-empty, and inconclusive when observed is empty.
func (e *Engine) Verify(ctx context.Context, executionID, expectedOutcome, observedOutcome string) (VerificationRecord, error) {
	if err := ctx.Err(); err != nil {
		return VerificationRecord{}, err
	}
	if strings.TrimSpace(executionID) == "" {
		return VerificationRecord{}, errors.New("execution ID is required")
	}

	id, err := newUUID()
	if err != nil {
		return VerificationRecord{}, fmt.Errorf("generate verification ID: %w", err)
	}

	now := time.Now().UTC()
	var result Result
	switch {
	case observedOutcome == "":
		result = ResultInconclusive
	case expectedOutcome == observedOutcome:
		result = ResultVerified
	default:
		result = ResultFailed
	}

	return VerificationRecord{
		VerificationID:  id,
		ExecutionID:     executionID,
		Method:          "outcome_comparison",
		ExpectedOutcome: strings.TrimSpace(expectedOutcome),
		ObservedOutcome: strings.TrimSpace(observedOutcome),
		Result:          result,
		VerifiedAt:      now,
	}, nil
}
