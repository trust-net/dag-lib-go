package repo

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/go-trust-net/common"
	"sync"
)

type DagNode struct {
	// parent node in the DAG
	Parent [64]byte
	// children nodes in the DAG
	Children [][64]byte
	// reference to Transaction ID in transaction table
	TxId [64]byte
	// Depth of the node in DAG
	Depth [8]byte
}

type DltDb interface {
	// get a transaction from transaction history (no entry == nil)
	GetTx(id [64]byte) *dto.Transaction
	// add a new transaction to transaction history (no duplicates, no updates)
	AddTx(tx *dto.Transaction) error
	// delete an existing transaction from transaction history (deleting a non-tip transaction will cause errors)
	DeleteTx(id [64]byte) error
	// get the shard's DAG node for given transaction Id (no entry == nil)
	GetShardDagNode(id [64]byte) *DagNode
	// get the submitter's DAG node for given transaction Id (no entry == nil)
	GetSubmitterDagNode(id [64]byte) *DagNode
	// get list of shards seen so far based on transaction history
	GetShards() []byte
	// get list of submitters seen so far based on transaction history
	GetSubmitters() []byte
	// get tip DAG nodes for sharder's DAG
	ShardTips(shardId []byte) []DagNode
	// get tip DAG nodes for submmiter's DAG
	SubmitterTips(submitterId []byte) []DagNode
}

type dltDb struct {
	txDb        db.Database
	shardDb     db.Database
	submitterDb db.Database
	dagDb       db.Database
	lock        sync.RWMutex
}

func (d *dltDb) GetTx(id [64]byte) *dto.Transaction {
	d.lock.Lock()
	defer d.lock.Unlock()
	// get serialized transactions from DB
	if data, err := d.txDb.Get(id[:]); err != nil {
		return nil
	} else {
		// deserialize the transaction read from DB
		tx := &dto.Transaction{}
		if err := tx.DeSerialize(data); err != nil {
			return nil
		}
		return tx
	}
}

func (d *dltDb) AddTx(tx *dto.Transaction) error {
	// save transaction
	var data []byte
	var err error
	if data, err = tx.Serialize(); err != nil {
		return err
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	// check for duplicate transaction
	id := tx.Id()
	if present, _ := d.txDb.Has(id[:]); present {
		return errors.New("duplicate transaction")
	}

	// save the transaction in DB
	if err = d.txDb.Put(id[:], data); err != nil {
		return err
	}

	// add the DAG node for the transaction to shard DAG db
	dagNode := DagNode{
		Parent: tx.ShardParent,
		TxId:   tx.Id(),
		Depth:  tx.ShardSeq,
	}
	if err = d.saveShardDagNode(&dagNode); err != nil {
		return err
	}

	// TBD: update the children of the parent DAG (if present)
	if parent := d.getShardDagNode(tx.ShardParent); parent != nil {
		parent.Children = append(parent.Children, tx.Id())
		if err := d.saveShardDagNode(parent); err != nil {
			return err
		}
	}

	return nil
}

func (d *dltDb) saveShardDagNode(node *DagNode) error {
	var data []byte
	var err error
	if data, err = common.Serialize(node); err != nil {
		return err
	}
	if err = d.shardDb.Put(node.TxId[:], data); err != nil {
		return err
	}
	return nil
}

func (d *dltDb) DeleteTx(id [64]byte) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if err := d.txDb.Delete(id[:]); err != nil {
		return err
	}
	return nil
}

func (d *dltDb) GetShardDagNode(id [64]byte) *DagNode {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.getShardDagNode(id)
}

func (d *dltDb) getShardDagNode(id [64]byte) *DagNode {
	// get serialized DAG node from DB
	if data, err := d.shardDb.Get(id[:]); err != nil {
		return nil
	} else {
		// deserialize the DAG node read from DB
		dagNode := &DagNode{}
		if err := common.Deserialize(data, dagNode); err != nil {
			return nil
		}
		return dagNode
	}
}

func (d *dltDb) GetSubmitterDagNode(id [64]byte) *DagNode {
	return nil
}

func (d *dltDb) GetShards() []byte {
	return nil
}

func (d *dltDb) GetSubmitters() []byte {
	return nil
}

func (d *dltDb) ShardTips(shardId []byte) []DagNode {
	return nil
}

func (d *dltDb) SubmitterTips(submitterId []byte) []DagNode {
	return nil
}

func NewDltDb(dbp db.DbProvider) (*dltDb, error) {
	return &dltDb{
		txDb:        dbp.DB("dlt_transactions"),
		shardDb:     dbp.DB("dlt_shards"),
		submitterDb: dbp.DB("dlt_submitters"),
		dagDb:       dbp.DB("dlt_dags"),
	}, nil
}
