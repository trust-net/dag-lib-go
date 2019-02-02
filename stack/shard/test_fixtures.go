// Copyright 2018-2019 The trust-net Authors
package shard

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
)

func SignedShardTransaction(payload string) (dto.Transaction, dto.Transaction) {
	tx := dto.TestSignedTransaction(payload)
	genesis := GenesisShardTx(tx.Request().ShardId)
	tx.Anchor().ShardParent = genesis.Id()
	return tx, genesis
}
