package stack

import (
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"testing"
)

func TestValidateSignature_ValidPayload(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _, _ := initMocksAndDb()
	// now switch the p2p Layer in DLT stack to real validation
	if p2pLayer, err := p2p.NewDEVp2pLayer(p2p.TestConfig(), stack.runner); err == nil {
		stack.p2p = p2pLayer
	} else {
		t.Errorf("Failed to instantiate real p2p layer: %s", err)
	}

	// create a transaction with correct submitter sequence as "nonce" value
	submitter := dto.TestSubmitter()
	newTx := submitter.NewTransaction(stack.Anchor(submitter.Id, submitter.Seq, submitter.LastTx), "test")
	if err := stack.Submit(newTx); err != nil {
		t.Errorf("Failed to submit transaction: %s", err)
	}
}

// test payload replay attack during transaction signature valiation
func TestValidateSignature_PayloadReplayInjection(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _, _ := initMocksAndDb()
	// now switch the p2p Layer in DLT stack to real validation
	if p2pLayer, err := p2p.NewDEVp2pLayer(p2p.TestConfig(), stack.runner); err == nil {
		stack.p2p = p2pLayer
	} else {
		t.Errorf("Failed to instantiate real p2p layer: %s", err)
	}
	// submit a valid transaction with original submitter
	submitter := dto.TestSubmitter()
	origTx := submitter.NewTransaction(stack.Anchor(submitter.Id, submitter.Seq, submitter.LastTx), "test")
	if err := stack.Submit(origTx); err != nil {
		t.Errorf("Failed to submit transaction: %s", err)
	}

	log.SetLogLevel(log.DEBUG)
	defer log.SetLogLevel(log.NONE)

	// now create a new transaction using a new anchor for submitter
	fakeTx := dto.NewTransaction(stack.Anchor(submitter.Id, submitter.Seq+1, origTx.Id()))

	// copy original transaction's payload to new transaction
	fakeTx.Self().Payload = origTx.Self().Payload

	// copy original transaction's payload signature to new transaction
	fakeTx.Self().Signature = origTx.Self().Signature

	// submit transaction, it should fail at payload signature validation
	if err := stack.Submit(fakeTx); err == nil {
		t.Errorf("Failed to detect payload replay attack")
	}
}
