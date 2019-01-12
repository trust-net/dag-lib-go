package endorsement

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"testing"
)

func TestInitiatization(t *testing.T) {
	var e Endorser
	var err error
	testDb := repo.NewMockDltDb()
	e, err = NewEndorser(testDb)
	if e.(*endorser) == nil || err != nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", e, err)
	}
	if e.(*endorser).db != testDb {
		t.Errorf("Layer does not have correct DB reference expected: %s, actual: %s", testDb, e.(*endorser).db)
	}
}

func TestTxHandler(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// send a mock transaction to endorser
	if res, err := e.Handle(dto.TestTransaction()); err != nil || res != SUCCESS {
		t.Errorf("Transacton handling failed: %s", err)
	}

	// validate that DltDb's AddTx method was called
	if testDb.AddTxCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}

	// validate the DLT DB's submitter was updated
	if testDb.UpdateSubmitterCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.UpdateSubmitterCount)
	}
}

func TestTxApprover(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// send a mock transaction to endorser
	if err := e.Approve(dto.TestTransaction()); err != nil {
		t.Errorf("Transacton approval failed: %s", err)
	}

	// validate the DLT DB's submitter was updated
	if testDb.UpdateSubmitterCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.UpdateSubmitterCount)
	}

	// validate that DltDb's AddTx method was NOT called during approval
	if testDb.AddTxCallCount != 0 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

// test that tx approver checks for double spending transaction
func TestTxApprover_DoubleSpending(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// create 2 double spending transactions using same submitter/seq/shard
	tx1 := dto.TestSignedTransaction("test data")
	tx2 := dto.TestSignedTransaction("test data")
	tx2.Anchor().Submitter = tx1.Anchor().Submitter
	tx2.Anchor().SubmitterSeq = tx1.Anchor().SubmitterSeq
	// make sure shard ID is same, for double spending
	tx2.Anchor().ShardId = tx1.Anchor().ShardId

	// send first transaction to endorser
	if err := e.Approve(tx1); err != nil {
		t.Errorf("Transacton approval failed: %s", err)
	}

	// send second transaction to endorser
	if err := e.Approve(tx2); err == nil {
		t.Errorf("Transacton approval did not fail for double spending")
	}

	// validate the DLT DB's submitter history was checked twice
	if testDb.GetSubmitterHistoryCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.GetSubmitterHistoryCount)
	}

	// validate the DLT DB's submitter update was called only once
	if testDb.UpdateSubmitterCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.UpdateSubmitterCount)
	}
}

// test that tx approver allows for relaxed sequence requirements
func TestTxApprover_RelaxedSequenceRequirements(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// create 2 double spending transactions using same submitter/seq, but different shard
	tx1 := dto.TestSignedTransaction("test data")
	tx2 := dto.TestSignedTransaction("test data")
	tx2.Anchor().Submitter = tx1.Anchor().Submitter
	tx2.Anchor().SubmitterSeq = tx1.Anchor().SubmitterSeq
	// make sure shard ID is different, for relaxed sequence requirement
	tx2.Anchor().ShardId = []byte("a different shard")

	// send first transaction to endorser
	if err := e.Approve(tx1); err != nil {
		t.Errorf("Transacton approval failed: %s", err)
	}

	// send second transaction to endorser
	if err := e.Approve(tx2); err != nil {
		t.Errorf("Transacton approval failed: %s", err)
	}

	// validate the DLT DB's submitter update was called twice
	if testDb.UpdateSubmitterCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.UpdateSubmitterCount)
	}
}

func TestTxHandlerSavesTransaction(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// send a transaction to endorser
	tx := dto.TestSignedTransaction("test payload")
	e.Handle(tx)

	// verify if transaction is saved into endorser's DB using Transaction's signature as key
	if present := e.db.GetTx(tx.Id()); present == nil {
		t.Errorf("Transacton handling did not save the transaction")
	}
}

func TestTxHandlerBadTransaction(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// send a nil transaction to endorser
	if res, err := e.Handle(nil); err == nil || res != ERR_INVALID {
		t.Errorf("Transacton handling did not check for nil transaction")
	}

	// send a duplicate transaction to endorser
	tx1 := dto.TestSignedTransaction("test payload")
	e.Handle(tx1)
	if res, err := e.Handle(tx1); err == nil || res != ERR_DUPLICATE {
		t.Errorf("Transacton handling did not check for duplicate transaction: %d == %s", res, err)
	}

	// validate that DltDb's AddTx method was called two times
	if testDb.AddTxCallCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}

	// validate the DLT DB's submitter update was called only once
	if testDb.UpdateSubmitterCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.UpdateSubmitterCount)
	}
}

// test that tx handler checks for double spending transaction
func TestTxHandler_DoubleSpending(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// create 2 double spending transactions using same submitter/seq/shard
	tx1 := dto.TestSignedTransaction("test data")
	tx2 := dto.TestSignedTransaction("test data")
	tx2.Anchor().Submitter = tx1.Anchor().Submitter
	tx2.Anchor().SubmitterSeq = tx1.Anchor().SubmitterSeq
	// make sure shard ID is same, for double spending
	tx2.Anchor().ShardId = tx1.Anchor().ShardId

	// send first transaction to endorser
	if _, err := e.Handle(tx1); err != nil {
		t.Errorf("Transacton handler failed: %s", err)
	}

	// send second transaction to endorser
	if res, err := e.Handle(tx2); err == nil || res != ERR_DOUBLE_SPEND {
		t.Errorf("Transacton handler did not fail for double spending")
	}

	// validate the DLT DB's submitter history was checked twice
	if testDb.GetSubmitterHistoryCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.GetSubmitterHistoryCount)
	}

	// validate that DltDb's AddTx method was called one times
	if testDb.AddTxCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}

	// validate the DLT DB's submitter update was called once
	if testDb.UpdateSubmitterCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.UpdateSubmitterCount)
	}
}

// test that tx handler allows for relaxed sequence requirements
func TestTxHandler_RelaxedSequenceRequirements(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// create 2 double spending transactions using same submitter/seq, but different shard
	tx1 := dto.TestSignedTransaction("test data")
	tx2 := dto.TestSignedTransaction("test data")
	tx2.Anchor().Submitter = tx1.Anchor().Submitter
	tx2.Anchor().SubmitterSeq = tx1.Anchor().SubmitterSeq
	// make sure shard ID is different, for relaxed sequence requirement
	tx2.Anchor().ShardId = []byte("a different shard")

	// send first transaction to endorser
	if _, err := e.Handle(tx1); err != nil {
		t.Errorf("Transacton approval failed: %s", err)
	}

	// send second transaction to endorser
	if _, err := e.Handle(tx2); err != nil {
		t.Errorf("Transacton approval failed: %s", err)
	}

	// validate that DltDb's AddTx method was called two times
	if testDb.AddTxCallCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}

	// validate the DLT DB's submitter update was called twice
	if testDb.UpdateSubmitterCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.UpdateSubmitterCount)
	}
}

// anchor method validates that submitter is using correct sequence and parent transaction in anchor request
func TestAnchor_ValidSubmitterRequest(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// pre-populate DLT DB with a submitter/seq transaction
	parent := dto.TestSignedTransaction("transaction 1")
	if err := testDb.UpdateSubmitter(parent); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	testDb.Reset()

	// create a new submitter anchor with pre-populated transaction as parent
	a := &dto.Anchor{
		Submitter:       parent.Anchor().Submitter,
		SubmitterSeq:    parent.Anchor().SubmitterSeq + 1,
		SubmitterLastTx: parent.Id(),
	}

	// send anchor for validation to endorser
	if err := e.Anchor(a); err != nil {
		t.Errorf("Anchor validation failed: %s", err)
	}

	// validate that submitter history was fetched for the parent and for current sequence
	if testDb.GetSubmitterHistoryCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.GetSubmitterHistoryCount)
	}
}

// anchor method validates that submitter is using correct parent transaction in anchor request
func TestAnchor_InvalidParent(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// pre-populate DLT DB with a submitter/seq transaction
	parent := dto.TestSignedTransaction("transaction 1")
	if err := testDb.UpdateSubmitter(parent); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	testDb.Reset()

	// create a new submitter anchor with pre-populated transaction as parent, but incorrect parent hash
	a := &dto.Anchor{
		Submitter:       parent.Anchor().Submitter,
		SubmitterSeq:    parent.Anchor().SubmitterSeq + 20,
		SubmitterLastTx: dto.RandomHash(),
	}

	// send anchor for validation to endorser
	if err := e.Anchor(a); err == nil {
		t.Errorf("Anchor validation did not check parent sequence")
	}

	// validate that submitter history was fetched for the parent but not for current sequence
	if testDb.GetSubmitterHistoryCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.GetSubmitterHistoryCount)
	}
}

// anchor method validates that submitter is using correct sequence in anchor request
func TestAnchor_UnexpectedSequence(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// pre-populate DLT DB with a submitter/seq transaction
	parent := dto.TestSignedTransaction("transaction 1")
	if err := testDb.UpdateSubmitter(parent); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	testDb.Reset()

	// create a new submitter anchor with pre-populated transaction as parent, but incorrect sequence
	a := &dto.Anchor{
		Submitter:       parent.Anchor().Submitter,
		SubmitterSeq:    parent.Anchor().SubmitterSeq + 20,
		SubmitterLastTx: parent.Id(),
	}

	// send anchor for validation to endorser
	if err := e.Anchor(a); err == nil {
		t.Errorf("Anchor validation did not check parent sequence")
	}

	// validate that submitter history was fetched for the parent but not for current sequence
	if testDb.GetSubmitterHistoryCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.GetSubmitterHistoryCount)
	}
}

// anchor method validates that submitter is not attempting double spending
func TestAnchor_DoubleSpending(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// pre-populate DLT DB with a parent/child transaction sequence
	parent := dto.TestSignedTransaction("transaction 1")
	if err := testDb.UpdateSubmitter(parent); err != nil {
		t.Errorf("Failed to update parent transaction: %s", err)
	}
	child := dto.TestSignedTransaction("test data")
	child.Anchor().Submitter = parent.Anchor().Submitter
	child.Anchor().SubmitterLastTx = parent.Id()
	child.Anchor().SubmitterSeq = parent.Anchor().SubmitterSeq + 1
	if err := testDb.UpdateSubmitter(child); err != nil {
		t.Errorf("Failed to update child transaction: %s", err)
	}
	testDb.Reset()

	// create a new submitter anchor with same submitter ID, Seq and Shard ID
	a := &dto.Anchor{
		Submitter:       child.Anchor().Submitter,
		SubmitterSeq:    child.Anchor().SubmitterSeq,
		SubmitterLastTx: parent.Id(),
		ShardId:         child.Anchor().ShardId,
	}

	// send anchor for validation to endorser
	if err := e.Anchor(a); err == nil {
		t.Errorf("Anchor validation did not check double spending")
	}

	// validate that submitter history was fetched twice, once for the parent and then for current sequence
	if testDb.GetSubmitterHistoryCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.GetSubmitterHistoryCount)
	}
}

// anchor method allows relaxed submitter sequence requirements
func TestAnchor_RelaxedSequenceRequirements(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// pre-populate DLT DB with a parent/child transaction sequence
	parent := dto.TestSignedTransaction("transaction 1")
	if err := testDb.UpdateSubmitter(parent); err != nil {
		t.Errorf("Failed to update parent transaction: %s", err)
	}
	child := dto.TestSignedTransaction("test data")
	child.Anchor().Submitter = parent.Anchor().Submitter
	child.Anchor().SubmitterLastTx = parent.Id()
	child.Anchor().SubmitterSeq = parent.Anchor().SubmitterSeq + 1
	if err := testDb.UpdateSubmitter(child); err != nil {
		t.Errorf("Failed to update child transaction: %s", err)
	}
	testDb.Reset()

	// create a new submitter anchor with same submitter ID, Seq but a different Shard ID
	a := &dto.Anchor{
		Submitter:       child.Anchor().Submitter,
		SubmitterSeq:    child.Anchor().SubmitterSeq,
		SubmitterLastTx: parent.Id(),
		ShardId:         []byte("a different shard"),
	}

	// send anchor for validation to endorser
	if err := e.Anchor(a); err != nil {
		t.Errorf("Anchor validation did not allow different shard with same sequence")
	}

	// validate that submitter history was fetched twice, once for the parent and then for current sequence
	if testDb.GetSubmitterHistoryCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.GetSubmitterHistoryCount)
	}
}
