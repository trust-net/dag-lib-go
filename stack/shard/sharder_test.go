package shard

import (
    "testing"
	"github.com/trust-net/dag-lib-go/db"
)

func TestInitiatization(t *testing.T) {
	var s Sharder
	var err error
	s, err = NewSharder(db.NewInMemDatabase())
	if s.(*sharder) == nil || err != nil {
		t.Errorf("Initiatization validation failed: %s, err: %s", s, err)
	}
}
