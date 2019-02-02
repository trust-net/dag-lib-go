// Copyright 2018-2019 The trust-net Authors
package consensus

import (
	"github.com/trust-net/dag-lib-go/db"
	"testing"
)

func TestInitiatization(t *testing.T) {
	var tr Trustee
	var err error
	tr, err = NewTrustee(db.NewInMemDatabase("test db"))
	if tr.(*trustee) != nil || err == nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", tr, err)
	}
}
