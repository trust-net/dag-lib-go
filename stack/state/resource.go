// Copyright 2019 The trust-net Authors
// Resource for world state
package state

import (
	"github.com/trust-net/dag-lib-go/common"
)

// A resource in the world state of a shard
type Resource struct {
	// key identifier for the resource (typically a UUID)
	Key []byte
	// identity of the owner of this resource
	Owner []byte
	// opaque serialized value of the resource
	Value []byte
}

func (r *Resource) Serialize() ([]byte, error) {
	return common.Serialize(r)
}

func (r *Resource) DeSerialize(data []byte) error {
	if err := common.Deserialize(data, r); err != nil {
		return err
	}
	return nil
}
