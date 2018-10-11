package consensus

import (
    "testing"
	"github.com/trust-net/dag-lib-go/db"
)

func TestInitiatization(t *testing.T) {
	var tr Trustee
	var err error
	tr, err = NewTrustee(db.NewInMemDatabase())
	if tr.(*trustee) != nil || err == nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", tr, err)
	}
}
