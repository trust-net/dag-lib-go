package shard

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
)

func SignedShardTransaction(payload string) (dto.Transaction, dto.Transaction) {
	tx := dto.TestSignedTransaction("test payload")
	genesis := GenesisShardTx(tx.Anchor().ShardId)
	tx.Anchor().ShardParent = genesis.Id()
	return tx, genesis
}
