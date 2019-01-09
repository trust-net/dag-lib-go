package repo

import (
	"github.com/trust-net/dag-lib-go/common"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"testing"
)

func TestInitiatization(t *testing.T) {
	var repo DltDb
	var err error
	testDb := db.NewInMemDbProvider()
	repo, err = NewDltDb(testDb)
	db := repo.(*dltDb)
	if db == nil || err != nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", repo, err)
	}
	if db.txDb != testDb.DB("dlt_transactions") {
		t.Errorf("Incorrect Transaction DB reference expected: %s, actual: %s", testDb.DB("dlt_transactions").Name(), db.txDb.Name())
	}
	if db.shardDAGsDb != testDb.DB("dlt_shard_dags") {
		t.Errorf("Incorrect Shards DB reference expected: %s, actual: %s", testDb.DB("dlt_shard_dags").Name(), db.shardDAGsDb.Name())
	}
	if db.shardTipsDb != testDb.DB("dlt_shard_tips") {
		t.Errorf("Incorrect Shards DB reference expected: %s, actual: %s", testDb.DB("dlt_shard_tips").Name(), db.shardTipsDb.Name())
	}
	if db.submitterDAGsDb != testDb.DB("dlt_submitter_dags") {
		t.Errorf("Incorrect Submitters DB reference expected: %s, actual: %s", testDb.DB("dlt_submitter_dags").Name(), db.submitterDAGsDb.Name())
	}
}

// test adding transaction
func TestAddTx(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx := dto.TestSignedTransaction("test data")
	txId := tx.Id()
	tx_ser, _ := tx.Serialize()

	// save transaction
	if err := repo.AddTx(tx); err != nil {
		t.Errorf("Failed to add transaction: %s", err)
	}

	// validate that transaction was added to Transaction DB correctly
	if got_tx, err := repo.txDb.Get(txId[:]); err != nil {
		t.Errorf("Error in checking transaction DB: %s", err)
	} else if string(got_tx) != string(tx_ser) {
		t.Errorf("Got incorrect transaction\nExpected: %x\nActual: %x", tx_ser, got_tx)
	}
}

// test adding transaction
func TestUpdateShard(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx := dto.TestSignedTransaction("test data")
	txId := tx.Id()

	// update shard with new transaction
	if err := repo.UpdateShard(tx); err != nil {
		t.Errorf("Failed to add transaction: %s", err)
	}

	// validate that shard DAG node was added for the transaction correctly
	if dagNode, _ := repo.shardDAGsDb.Get(txId[:]); dagNode == nil {
		t.Errorf("Did not save DAG node in shard DB")
	}

	// validate that shard tips was updated for the transaction correctly
	if _, err := repo.shardTipsDb.Get(tx.Anchor().ShardId); err != nil {
		t.Errorf("Error in checking shard tips: %s", err)
	}
}

// test updating sequential transactions
func TestUpdateShardInSequence(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx1 := dto.TestSignedTransaction("test data")
	tx2 := dto.TestSignedTransaction("test data")
	tx2.Anchor().ShardParent = tx1.Id()
	tx2.Anchor().ShardSeq = tx1.Anchor().ShardSeq

	// update shard with transaction sequence
	if err := repo.UpdateShard(tx1); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	if err := repo.UpdateShard(tx2); err != nil {
		t.Errorf("Failed to add 2nd transaction: %s", err)
	}

	// validate that shard DAG node was added for 1st transaction correctly
	if node := repo.GetShardDagNode(tx1.Id()); node == nil {
		t.Errorf("Did not save DAG node in shard DB for 1st transaction")
	} else if node.TxId != tx1.Id() {
		t.Errorf("Did not update node's TxId for 1st transaction")
	} else if node.Parent != tx1.Anchor().ShardParent {
		t.Errorf("Did not update node's parent for 1st transaction")
	}

	// validate that shard DAG node was added for 2nd transaction correctly
	if node := repo.GetShardDagNode(tx2.Id()); node == nil {
		t.Errorf("Did not save DAG node in shard DB for 2nd transaction")
	} else if node.TxId != tx2.Id() {
		t.Errorf("Did not update node's TxId for 2nd transaction")
	} else if node.Parent != tx1.Id() {
		t.Errorf("Did not update node's parent for 2nd transaction")
	}
}

// test shard DAG update during adding transaction
func TestAddTxShardDagUpdate(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	parent := dto.TestSignedTransaction("test data")
	child1 := dto.TestSignedTransaction("test data")
	child1.Anchor().ShardParent = parent.Id()
	child2 := dto.TestSignedTransaction("test data")
	child2.Anchor().ShardParent = parent.Id()

	// save transactions
	repo.AddTx(parent)
	repo.UpdateShard(parent)
	repo.AddTx(child1)
	repo.UpdateShard(child1)
	repo.AddTx(child2)
	repo.UpdateShard(child2)

	// validate that shard DAG node was added for the transactions correctly
	if parentNode := repo.GetShardDagNode(parent.Id()); parentNode == nil {
		t.Errorf("Did not save DAG node in shard DB")
	} else {
		// validate that children nodes were added correctly for parent's DAG node
		if len(parentNode.Children) != 2 {
			t.Errorf("Incorrect children count: %d", len(parentNode.Children))
		} else {
			if parentNode.Children[0] != child1.Id() {
				t.Errorf("Incorrect 1st child\nExpected: %x\nActual: %x", child1.Id(), parentNode.Children[0])
			}
			if parentNode.Children[1] != child2.Id() {
				t.Errorf("Incorrect 2nd child\nExpected: %x\nActual: %x", child2.Id(), parentNode.Children[1])
			}
		}
	}
}

// test shard tips update during adding transaction
func TestAddTxShardTipsUpdate(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	parent := dto.TestSignedTransaction("test data")
	child1 := dto.TestSignedTransaction("test data")
	child1.Anchor().ShardParent = parent.Id()
	child2 := dto.TestSignedTransaction("test data")
	child2.Anchor().ShardParent = parent.Id()

	// add a parent transaction
	repo.AddTx(parent)
	repo.UpdateShard(parent)
	// validate that shard tip was added for the transactions correctly
	tips := repo.ShardTips(parent.Anchor().ShardId)
	if len(tips) != 1 {
		t.Errorf("Incorrect number of tips: %d", len(tips))
	} else if tips[0] != parent.Id() {
		t.Errorf("Incorrect parent tip\nExpected: %x\nActual: %x", parent.Id(), tips[0])
	}

	// now add 2 child transactions for same parent
	repo.AddTx(child1)
	repo.UpdateShard(child1)
	repo.AddTx(child2)
	repo.UpdateShard(child2)

	// validate that shard tip was updated for the transactions correctly
	tips = repo.ShardTips(parent.Anchor().ShardId)
	if len(tips) != 2 {
		t.Errorf("Incorrect number of tips: %d", len(tips))
	} else {
		if tips[0] != child1.Id() {
			t.Errorf("Incorrect 1st tip\nExpected: %x\nActual: %x", child1.Id(), tips[0])
		}
		if tips[1] != child2.Id() {
			t.Errorf("Incorrect 2nd tip\nExpected: %x\nActual: %x", child2.Id(), tips[1])
		}
	}
}

// test shard tips coalescing during adding transaction
func TestAddTxShardTipsCoalesce(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	parent := dto.TestSignedTransaction("test data")
	uncle := dto.TestSignedTransaction("test data")
	child := dto.TestSignedTransaction("test data")
	child.Anchor().ShardParent = parent.Id()
	child.Anchor().ShardUncles = [][64]byte{uncle.Id()}

	// add parent transaction
	repo.AddTx(parent)
	repo.UpdateShard(parent)
	// add uncle transaction
	repo.AddTx(uncle)
	repo.UpdateShard(uncle)
	// validate that shard tip was added for the transactions correctly
	tips := repo.ShardTips(parent.Anchor().ShardId)
	if len(tips) != 2 {
		t.Errorf("Incorrect number of tips: %d", len(tips))
	}

	// now add child transactions for parent / uncle
	repo.AddTx(child)
	repo.UpdateShard(child)

	// validate that shard tip was updated for the transactions correctly
	tips = repo.ShardTips(parent.Anchor().ShardId)
	if len(tips) != 1 {
		t.Errorf("Incorrect number of tips: %d", len(tips))
	} else if tips[0] != child.Id() {
		t.Errorf("Incorrect 1st tip\nExpected: %x\nActual: %x", child.Id(), tips[0])
	}
}

// test adding duplicate transaction
func TestAddDuplicateTx(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx := dto.TestSignedTransaction("test data")

	// save transaction twice
	repo.AddTx(tx)
	if err := repo.AddTx(tx); err == nil {
		t.Errorf("Failed to detect duplicate transaction")
	}
}

// test adding transaction with no parent (DB will add, assumption is that sharding or endorser layer check for orphan)
func TestAddOrphanTx(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx := dto.TestSignedTransaction("test data")

	// make transaction orphan
	tx.Anchor().ShardSeq = 0x02
	parent := []byte("some random parent")
	for i := 0; i < len(tx.Anchor().ShardParent) && i < len(parent); i++ {
		tx.Anchor().ShardParent[i] = parent[i]
	}

	// save the orphaned transaction
	if err := repo.AddTx(tx); err != nil {
		t.Errorf("Failed to add orphan transaction: %s", err)
	}
}

// test getting a transaction after adding
func TestGetTx(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())

	// first try to get a transaction without adding
	tx := dto.TestSignedTransaction("test data")
	if repo.GetTx(tx.Id()) != nil {
		t.Errorf("Did not expect a transaction that is not saved yet!!!")
	}

	// now save transaction and then get, this time it should work
	repo.AddTx(tx)
	got_tx := repo.GetTx(tx.Id())
	if got_tx == nil {
		t.Errorf("Did not get a saved transaction!!!")
	} else if got_tx.Id() != tx.Id() {
		t.Errorf("Got incorrect transaction\nExpected: %x\nActual: %x", tx.Id(), got_tx.Id())
	}
}

// test deleting a transaction
func TestDeleteTx(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())

	// first save a transaction
	tx := dto.TestSignedTransaction("test data")
	txId := tx.Id()
	repo.AddTx(tx)

	// now delete transaction
	if err := repo.DeleteTx(tx.Id()); err != nil {
		t.Errorf("Failed to delete transaction: %s", err)
	}

	// validate that transaction was deleted from Transaction DB correctly
	if got_tx, _ := repo.txDb.Get(txId[:]); got_tx != nil {
		t.Errorf("Transaction not deleted from DB")
	}
}

// test get shard DAG after adding transaction
func TestGetShardDagNode(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx := dto.TestSignedTransaction("test data")

	// save transaction
	repo.AddTx(tx)
	repo.UpdateShard(tx)

	// validate that can get shard DAG node after adding transaction
	if dagNode := repo.GetShardDagNode(tx.Id()); dagNode == nil {
		t.Errorf("Cannot get DAG node in shard DB")
	}
}

// test update to submitter DAG
func TestUpdateSubmitter(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx := dto.TestSignedTransaction("test data")
	txId := tx.Id()

	// update submitter with new transaction
	if err := repo.UpdateSubmitter(tx); err != nil {
		t.Errorf("Failed to add transaction: %s", err)
	}

	// validate that submitter DAG node was added for the transaction correctly
	if dagNode, _ := repo.submitterDAGsDb.Get(txId[:]); dagNode == nil {
		t.Errorf("Did not save DAG node in shard DB")
	}
	// validate that submitter history was updated for the transaction correctly
	key := []byte{}
	key = append(key, tx.Anchor().Submitter...)
	key = append(key, ':')
	key = append(key, common.Uint64ToBytes(tx.Anchor().SubmitterSeq)...)
	if data, err := repo.submitterHistoryDb.Get(key); err != nil || string(data) != string(txId[:]) {
		t.Errorf("Did not update submitter history in shard DB")
	}
}

// test updating sequential transactions
func TestUpdateSubmitterInSequence(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx1 := dto.TestSignedTransaction("test data")
	tx2 := dto.TestSignedTransaction("test data")
	tx2.Anchor().Submitter = tx1.Anchor().Submitter
	tx2.Anchor().SubmitterLastTx = tx1.Id()
	tx2.Anchor().SubmitterSeq = tx1.Anchor().SubmitterSeq + 1

	// update submitter with transaction sequence
	if err := repo.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	if err := repo.UpdateSubmitter(tx2); err != nil {
		t.Errorf("Failed to add 2nd transaction: %s", err)
	}

	// validate that submitter DAG node was added for 1st transaction correctly
	if node := repo.GetSubmitterDagNode(tx1.Id()); node == nil {
		t.Errorf("Did not save DAG node in shard DB for 1st transaction")
	} else if node.TxId != tx1.Id() {
		t.Errorf("Did not update node's TxId for 1st transaction")
	} else if node.Parent != tx1.Anchor().SubmitterLastTx {
		t.Errorf("Did not update node's parent for 1st transaction")
	} else if len(node.Children) == 0 {
		t.Errorf("Did not update parent node's children list")
	} else if node.Children[0] != tx2.Id() {
		t.Errorf("Did add child node to parent node's children list")
	}

	// validate that shard DAG node was added for 2nd transaction correctly
	if node := repo.GetSubmitterDagNode(tx2.Id()); node == nil {
		t.Errorf("Did not save DAG node in shard DB for 2nd transaction")
	} else if node.TxId != tx2.Id() {
		t.Errorf("Did not update node's TxId for 2nd transaction")
	} else if node.Parent != tx1.Id() {
		t.Errorf("Did not update node's parent for 2nd transaction")
	}
}

// test submitter history when present
func TestGetSubmitterHistory_Present(t *testing.T) {
	repo, _ := NewDltDb(db.NewInMemDbProvider())
	tx := dto.TestSignedTransaction("test data")

	// update submitter with new transaction
	if err := repo.UpdateSubmitter(tx); err != nil {
		t.Errorf("Failed to add transaction: %s", err)
	}

	// fetch submitter history for the added transaction's submitter and sequence
	if node := repo.GetSubmitterHistory(tx.Anchor().Submitter, tx.Anchor().SubmitterSeq); node == nil {
		t.Errorf("Failed to fetch submitter history")
	} else if node.TxId != tx.Id() {
		t.Errorf("fetched incorrect submitter history")
	}

}
