package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
)

func TestPostgresStoreLifecycleAndIdempotency(t *testing.T) {
	if os.Getenv("NAEOS_POSTGRES_INTEGRATION") != "1" { t.Skip("set NAEOS_POSTGRES_INTEGRATION=1 to run PostgreSQL integration test") }
	dsn:=os.Getenv("DATABASE_URL")
	if dsn=="" { t.Fatal("DATABASE_URL must be set for PostgreSQL integration test") }
	ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel()
	if err:=Up(ctx,dsn,"../../migrations");err!=nil{t.Fatalf("apply migrations: %v",err)}
	pool,err:=OpenPool(ctx,dsn);if err!=nil{t.Fatalf("open pool: %v",err)};defer pool.Close()
	store:=New(pool)
	clientID:="integration-"+time.Now().UTC().Format("20060102150405.000000000")
	input:=httpapi.RequestInput{Purpose:"persistence integration",ExecutionMode:"analysis_only",DesiredOutputs:[]string{"trace"}}

	first,replay,err:=store.Create(ctx,clientID,"idem-key",input,"correlation-test")
	if err!=nil{t.Fatalf("create: %v",err)}
	if replay || first.Status!=httpapi.StatusAccepted {t.Fatalf("first create replay=%v status=%s",replay,first.Status)}
	second,replay,err:=store.Create(ctx,clientID,"idem-key",input,"correlation-test")
	if err!=nil{t.Fatalf("idempotent replay: %v",err)}
	if !replay || second.RequestID!=first.RequestID {t.Fatalf("replay=%v id=%s want id=%s",replay,second.RequestID,first.RequestID)}
	conflicting:=input;conflicting.Purpose="changed payload"
	if _,_,err=store.Create(ctx,clientID,"idem-key",conflicting,"correlation-test");!errors.Is(err,httpapi.ErrIdempotencyConflict){t.Fatalf("want idempotency conflict, got %v",err)}
	if _,err=store.Get(ctx,"another-client",first.RequestID);!errors.Is(err,httpapi.ErrNotFound){t.Fatalf("cross-client access err=%v",err)}
	cancelled,err:=store.Cancel(ctx,clientID,first.RequestID,"correlation-cancel")
	if err!=nil || cancelled.Status!=httpapi.StatusCancelled{t.Fatalf("cancelled=%+v err=%v",cancelled,err)}
	if _,err=store.Cancel(ctx,clientID,first.RequestID,"correlation-cancel");err!=nil{t.Fatalf("idempotent cancel: %v",err)}

	// Concurrent cancellation requests must serialize against the row lock and
	// append exactly one lifecycle transition while remaining idempotent.
	raceRequest, _, err := store.Create(ctx, clientID+"-race", "", input, "correlation-race")
	if err != nil {
		t.Fatalf("create cancellation-race request: %v", err)
	}
	const cancelCallCount = 12
	var cancelWG sync.WaitGroup
	cancelErrors := make(chan error, cancelCallCount)
	for i := 0; i < cancelCallCount; i++ {
		cancelWG.Add(1)
		go func() {
			defer cancelWG.Done()
			result, cancelErr := store.Cancel(ctx, clientID+"-race", raceRequest.RequestID, "correlation-race-cancel")
			if cancelErr != nil {
				cancelErrors <- cancelErr
				return
			}
			if result.Status != httpapi.StatusCancelled {
				cancelErrors <- errors.New("concurrent cancel returned non-cancelled status")
			}
		}()
	}
	cancelWG.Wait()
	close(cancelErrors)
	for cancelErr := range cancelErrors {
		t.Errorf("concurrent cancellation: %v", cancelErr)
	}
	var transitionCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM request_transitions WHERE request_id=$1::uuid AND event_type='request.cancelled'", raceRequest.RequestID).Scan(&transitionCount); err != nil {
		t.Fatalf("count cancellation transitions: %v", err)
	}
	if transitionCount != 1 {
		t.Fatalf("concurrent cancel created %d transitions, want exactly 1", transitionCount)
	}
	finalRequest, err := store.Get(ctx, clientID+"-race", raceRequest.RequestID)
	if err != nil || finalRequest.Status != httpapi.StatusCancelled {
		t.Fatalf("final race request status=%s err=%v", finalRequest.Status, err)
	}

	attempt,err:=store.CreateAttempt(ctx,first.RequestID);if err!=nil{t.Fatalf("create attempt: %v",err)}
	if attempt.Status!="pending" || attempt.AttemptNumber!=1{t.Fatalf("unexpected attempt: %+v",attempt)}
	if err=store.StartAttempt(ctx,first.RequestID,attempt.AttemptNumber,"worker-integration");err!=nil{t.Fatalf("start attempt: %v",err)}
	if err=store.FinishAttempt(ctx,first.RequestID,attempt.AttemptNumber,"failed","transient","simulated integration failure");err!=nil{t.Fatalf("finish attempt: %v",err)}
	if err=store.FinishAttempt(ctx,first.RequestID,attempt.AttemptNumber,"succeeded","","");!errors.Is(err,httpapi.ErrConflict){t.Fatalf("terminal transition must be one-shot, got %v",err)}
}
