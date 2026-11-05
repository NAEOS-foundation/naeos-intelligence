// Package postgres implements durable request persistence.
package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/governance"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct { pool *pgxpool.Pool }
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil { return "", err }
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[:8]+"-"+h[8:12]+"-"+h[12:16]+"-"+h[16:20]+"-"+h[20:], nil
}

func validUUID(v string) bool {
	if len(v)!=36 || v[8]!='-' || v[13]!='-' || v[18]!='-' || v[23]!='-' { return false }
	return func() bool { _,err:=hex.DecodeString(strings.ReplaceAll(v,"-",""));return err==nil }()
}

func digest(input httpapi.RequestInput) (string, error) {
	raw, err := json.Marshal(input); if err != nil { return "", err }
	sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]), nil
}

func (s *Store) Create(ctx context.Context, clientID, key string, input httpapi.RequestInput, correlationID string) (httpapi.Request, bool, error) {
	d, err := digest(input); if err != nil { return httpapi.Request{}, false, err }
	id, err := newUUID(); if err != nil { return httpapi.Request{}, false, err }
	transitionID, err := newUUID(); if err != nil { return httpapi.Request{}, false, err }
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{}); if err != nil { return httpapi.Request{}, false, err }
	defer tx.Rollback(ctx)
	var idem, payloadDigest any
	if key != "" { idem = key; payloadDigest = d }
	contextSources, err := json.Marshal(input.ContextSources); if err != nil { return httpapi.Request{}, false, err }
	desiredOutputs, err := json.Marshal(input.DesiredOutputs); if err != nil { return httpapi.Request{}, false, err }
	metadata, err := json.Marshal(input.Metadata); if err != nil { return httpapi.Request{}, false, err }
	if key != "" {
		tag, err := tx.Exec(ctx, `INSERT INTO intelligence_requests
			(id,client_id,purpose,execution_mode,status,context_sources,desired_outputs,metadata,idempotency_key,payload_digest)
			VALUES ($1,$2,$3,$4,'accepted',$5,$6,$7,$8,$9)
			ON CONFLICT (client_id,idempotency_key) DO NOTHING`,
			id,clientID,input.Purpose,input.ExecutionMode,contextSources,desiredOutputs,metadata,idem,payloadDigest)
		if err != nil { return httpapi.Request{}, false, err }
		if tag.RowsAffected() == 0 {
			var existingID, existingDigest string
			err := tx.QueryRow(ctx,`SELECT id::text,payload_digest FROM intelligence_requests WHERE client_id=$1 AND idempotency_key=$2 FOR UPDATE`,clientID,key).Scan(&existingID,&existingDigest)
			if err != nil { return httpapi.Request{}, false, err }
			if strings.TrimSpace(existingDigest) != d { return httpapi.Request{}, false, httpapi.ErrIdempotencyConflict }
			req, err := readRequest(ctx,tx,clientID,existingID); if err != nil { return httpapi.Request{}, false, err }
			if err := tx.Commit(ctx); err != nil { return httpapi.Request{}, false, err }
			return req, true, nil
		}
	} else {
		_, err = tx.Exec(ctx,`INSERT INTO intelligence_requests
			(id,client_id,purpose,execution_mode,status,context_sources,desired_outputs,metadata,idempotency_key,payload_digest)
			VALUES ($1,$2,$3,$4,'accepted',$5,$6,$7,NULL,NULL)`,
			id,clientID,input.Purpose,input.ExecutionMode,contextSources,desiredOutputs,metadata)
		if err != nil { return httpapi.Request{}, false, err }
	}
	_, err = tx.Exec(ctx,`INSERT INTO request_transitions
		(id,request_id,from_status,to_status,event_type,actor_type,actor_id,reason,correlation_id)
		VALUES ($1,$2,NULL,'accepted','request.accepted','client',$3,NULL,$4)`,transitionID,id,clientID,nullIfBlank(correlationID))
	if err != nil { return httpapi.Request{}, false, err }
	req, err := readRequest(ctx,tx,clientID,id); if err != nil { return httpapi.Request{}, false, err }
	if err := tx.Commit(ctx); err != nil { return httpapi.Request{}, false, err }
	return req, false, nil
}

func nullIfBlank(v string) any { if strings.TrimSpace(v)=="" { return nil }; return v }
type queryer interface { QueryRow(context.Context,string,...any) pgx.Row }

func readRequest(ctx context.Context, q queryer, clientID, requestID string) (httpapi.Request,error) {
	if !validUUID(requestID) { return httpapi.Request{},httpapi.ErrNotFound }
	var req httpapi.Request
	var id string
	err := q.QueryRow(ctx,`SELECT id::text,status,purpose,execution_mode,created_at,updated_at,client_id
		FROM intelligence_requests WHERE client_id=$1 AND id=$2::uuid`,clientID,requestID).
		Scan(&id,&req.Status,&req.Purpose,&req.ExecutionMode,&req.CreatedAt,&req.UpdatedAt,&req.ClientID)
	if errors.Is(err,pgx.ErrNoRows) { return httpapi.Request{},httpapi.ErrNotFound }
	if err != nil { return httpapi.Request{},err }
	req.RequestID=id; req.CreatedAt=req.CreatedAt.UTC(); req.UpdatedAt=req.UpdatedAt.UTC()
	return req,nil
}

func (s *Store) Get(ctx context.Context, clientID, requestID string) (httpapi.Request,error) {
	return readRequest(ctx,s.pool,clientID,requestID)
}

func (s *Store) Cancel(ctx context.Context, clientID, requestID, correlationID string) (httpapi.Request,error) {
	if !validUUID(requestID) { return httpapi.Request{},httpapi.ErrNotFound }
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{}); if err!=nil{return httpapi.Request{},err}
	defer tx.Rollback(ctx)
	var status httpapi.Status
	err=tx.QueryRow(ctx,`SELECT status FROM intelligence_requests WHERE client_id=$1 AND id=$2::uuid FOR UPDATE`,clientID,requestID).Scan(&status)
	if errors.Is(err,pgx.ErrNoRows){return httpapi.Request{},httpapi.ErrNotFound}
	if err!=nil{return httpapi.Request{},err}
	switch status {
	case httpapi.StatusAccepted,httpapi.StatusQueued,httpapi.StatusRequiresApproval:
		transitionID,idErr:=newUUID();if idErr!=nil{return httpapi.Request{},idErr}
		_,err=tx.Exec(ctx,`UPDATE intelligence_requests SET status='cancelled',updated_at=now() WHERE client_id=$1 AND id=$2::uuid`,clientID,requestID)
		if err!=nil{return httpapi.Request{},err}
		_,err=tx.Exec(ctx,`INSERT INTO request_transitions(id,request_id,from_status,to_status,event_type,actor_type,actor_id,reason,correlation_id)
			VALUES($1,$2::uuid,$3,'cancelled','request.cancelled','client',$4,NULL,$5)`,transitionID,requestID,status,clientID,nullIfBlank(correlationID))
		if err!=nil{return httpapi.Request{},err}
	case httpapi.StatusCancelled:
	default:return httpapi.Request{},httpapi.ErrConflict
	}
	req,err:=readRequest(ctx,tx,clientID,requestID);if err!=nil{return httpapi.Request{},err}
	if err=tx.Commit(ctx);err!=nil{return httpapi.Request{},err}
	return req,nil
}

func (s *Store) Close() { s.pool.Close() }
func (s *Store) Ping(ctx context.Context) error {
	if err:=s.pool.Ping(ctx);err!=nil{return fmt.Errorf("postgres ping: %w",err)}
	return nil
}

type Attempt struct {
	ID string
	RequestID string
	AttemptNumber int32
	Status string
	StartedAt *time.Time
	FinishedAt *time.Time
	FailureCategory string
	FailureDetail string
	WorkerID string
	CreatedAt time.Time
}

func (s *Store) CreateAttempt(ctx context.Context, requestID string) (Attempt,error) {
	if !validUUID(requestID){return Attempt{},httpapi.ErrNotFound}
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{});if err!=nil{return Attempt{},err}
	defer tx.Rollback(ctx)
	var exists bool
	err=tx.QueryRow(ctx,`SELECT true FROM intelligence_requests WHERE id=$1::uuid FOR UPDATE`,requestID).Scan(&exists)
	if errors.Is(err,pgx.ErrNoRows){return Attempt{},httpapi.ErrNotFound};if err!=nil{return Attempt{},err}
	var next int32
	err=tx.QueryRow(ctx,`SELECT COALESCE(MAX(attempt_number),0)+1 FROM request_attempts WHERE request_id=$1::uuid`,requestID).Scan(&next)
	if err!=nil{return Attempt{},err}
	id,err:=newUUID();if err!=nil{return Attempt{},err}
	var a Attempt
	err=tx.QueryRow(ctx,`INSERT INTO request_attempts(id,request_id,attempt_number,status)
		VALUES($1,$2::uuid,$3,'pending') RETURNING id::text,request_id::text,attempt_number,status,created_at`,id,requestID,next).
		Scan(&a.ID,&a.RequestID,&a.AttemptNumber,&a.Status,&a.CreatedAt)
	if err!=nil{return Attempt{},err}
	if err=tx.Commit(ctx);err!=nil{return Attempt{},err}
	a.CreatedAt=a.CreatedAt.UTC();return a,nil
}

func (s *Store) StartAttempt(ctx context.Context, requestID string, attemptNumber int32, workerID string) error {
	if !validUUID(requestID){return httpapi.ErrNotFound}
	if strings.TrimSpace(workerID)=="" || len(workerID)>255{return errors.New("worker ID must contain 1 to 255 characters")}
	tag,err:=s.pool.Exec(ctx,`UPDATE request_attempts SET status='running',started_at=now(),worker_id=$3
		WHERE request_id=$1::uuid AND attempt_number=$2 AND status='pending'`,requestID,attemptNumber,workerID)
	if err!=nil{return err};if tag.RowsAffected()!=1{return httpapi.ErrConflict};return nil
}

func (s *Store) FinishAttempt(ctx context.Context, requestID string, attemptNumber int32, outcome, failureCategory, failureDetail string) error {
	if !validUUID(requestID){return httpapi.ErrNotFound}
	switch outcome {case "succeeded","failed","cancelled":default:return errors.New("invalid terminal attempt outcome")}
	if len(failureCategory)>128 || len(failureDetail)>4000{return errors.New("failure metadata exceeds allowed length")}
	if outcome!="failed" && (failureCategory!="" || failureDetail!=""){return errors.New("failure metadata only allowed for failed outcome")}
	tag,err:=s.pool.Exec(ctx,`UPDATE request_attempts SET status=$3,finished_at=now(),failure_category=$4,failure_detail=$5
		WHERE request_id=$1::uuid AND attempt_number=$2 AND status='running'`,requestID,attemptNumber,outcome,nullIfBlank(failureCategory),nullIfBlank(failureDetail))
	if err!=nil{return err};if tag.RowsAffected()!=1{return httpapi.ErrConflict};return nil
}

func (s *Store) RecordApprovalDecision(ctx context.Context, clientID, requestID, decision, approvedBy, reason string) (governance.ApprovalDecision, error) {
	return governance.ApprovalDecision{}, errors.New("durable approval persistence not implemented in postgres store")
}
