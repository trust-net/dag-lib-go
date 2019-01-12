package repo

import (
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
)

type mockDb struct {
	GetTxCallCount               int
	AddTxCallCount               int
	UpdateShardCount             int
	UpdateSubmitterCount         int
	DeleteTxCallCount            int
	GetDagNodeCallCount          int
	GetShardDagNodeCallCount     int
	GetSubmitterDagNodeCallCount int
	GetSubmitterHistoryCount     int
	GetShardsCallCount           int
	GetSubmittersCallCount       int
	ShardTipsCallCount           int
	SubmitterTipsCallCount       int
	db                           DltDb
}

func (d *mockDb) GetTx(id [64]byte) dto.Transaction {
	d.GetTxCallCount += 1
	return d.db.GetTx(id)
}

func (d *mockDb) AddTx(tx dto.Transaction) error {
	d.AddTxCallCount += 1
	return d.db.AddTx(tx)
}

func (d *mockDb) UpdateShard(tx dto.Transaction) error {
	d.UpdateShardCount += 1
	return d.db.UpdateShard(tx)
}

func (d *mockDb) UpdateSubmitter(tx dto.Transaction) error {
	d.UpdateSubmitterCount += 1
	return d.db.UpdateSubmitter(tx)
}

func (d *mockDb) DeleteTx(id [64]byte) error {
	d.DeleteTxCallCount += 1
	return d.db.DeleteTx(id)
}

func (d *mockDb) GetShardDagNode(id [64]byte) *DagNode {
	d.GetShardDagNodeCallCount += 1
	return d.db.GetShardDagNode(id)
}
//
//func (d *mockDb) GetSubmitterDagNode(id [64]byte) *DagNode {
//	d.GetSubmitterDagNodeCallCount += 1
//	return d.db.GetSubmitterDagNode(id)
//}

func (d *mockDb) GetSubmitterHistory(id []byte, seq uint64) *SubmitterHistory {
	d.GetSubmitterHistoryCount += 1
	return d.db.GetSubmitterHistory(id, seq)
}

func (d *mockDb) GetShards() []byte {
	d.GetShardsCallCount += 1
	return d.db.GetShards()
}

func (d *mockDb) GetSubmitters() []byte {
	d.GetSubmittersCallCount += 1
	return d.db.GetSubmitters()
}

func (d *mockDb) ShardTips(shardId []byte) [][64]byte {
	d.ShardTipsCallCount += 1
	return d.db.ShardTips(shardId)
}

func (d *mockDb) SubmitterTips(submitterId []byte) []DagNode {
	d.SubmitterTipsCallCount += 1
	return d.db.SubmitterTips(submitterId)
}

func (d *mockDb) Reset() {
	*d = mockDb{db: d.db}
}

func NewMockDltDb() *mockDb {
	db, _ := NewDltDb(db.NewInMemDbProvider())
	return &mockDb{
		db: db,
	}
}
