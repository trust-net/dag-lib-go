// Copyright 2018-2019 The trust-net Authors
package repo

import (
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
)

type MockDltDb struct {
	GetTxCallCount               int
	FlushShardCount              int
	ReplaceSubmitterCount        int
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

func (d *MockDltDb) ReplaceSubmitter(tx dto.Transaction) error {
	d.ReplaceSubmitterCount += 1
	return d.db.ReplaceSubmitter(tx)
}

func (d *MockDltDb) FlushShard(shardId []byte) error {
	d.FlushShardCount += 1
	return d.db.FlushShard(shardId)
}

func (d *MockDltDb) GetTx(id [64]byte) dto.Transaction {
	d.GetTxCallCount += 1
	return d.db.GetTx(id)
}

func (d *MockDltDb) AddTx(tx dto.Transaction) error {
	d.AddTxCallCount += 1
	return d.db.AddTx(tx)
}

func (d *MockDltDb) UpdateShard(tx dto.Transaction) error {
	d.UpdateShardCount += 1
	return d.db.UpdateShard(tx)
}

func (d *MockDltDb) UpdateSubmitter(tx dto.Transaction) error {
	d.UpdateSubmitterCount += 1
	return d.db.UpdateSubmitter(tx)
}

func (d *MockDltDb) DeleteTx(id [64]byte) error {
	d.DeleteTxCallCount += 1
	return d.db.DeleteTx(id)
}

func (d *MockDltDb) GetShardDagNode(id [64]byte) *DagNode {
	d.GetShardDagNodeCallCount += 1
	return d.db.GetShardDagNode(id)
}

//
//func (d *mockDb) GetSubmitterDagNode(id [64]byte) *DagNode {
//	d.GetSubmitterDagNodeCallCount += 1
//	return d.db.GetSubmitterDagNode(id)
//}

func (d *MockDltDb) GetSubmitterHistory(id []byte, seq uint64) *SubmitterHistory {
	d.GetSubmitterHistoryCount += 1
	return d.db.GetSubmitterHistory(id, seq)
}

func (d *MockDltDb) GetShards() []byte {
	d.GetShardsCallCount += 1
	return d.db.GetShards()
}

func (d *MockDltDb) GetSubmitters() []byte {
	d.GetSubmittersCallCount += 1
	return d.db.GetSubmitters()
}

func (d *MockDltDb) ShardTips(shardId []byte) [][64]byte {
	d.ShardTipsCallCount += 1
	return d.db.ShardTips(shardId)
}

func (d *MockDltDb) SubmitterTips(submitterId []byte) []DagNode {
	d.SubmitterTipsCallCount += 1
	return d.db.SubmitterTips(submitterId)
}

func (d *MockDltDb) Reset() {
	*d = MockDltDb{db: d.db}
}

func NewMockDltDb() *MockDltDb {
	db, _ := NewDltDb(db.NewInMemDbProvider())
	return &MockDltDb{
		db: db,
	}
}
