package endorsement

import (
    "testing"
	"github.com/trust-net/dag-lib-go/db"
)

func TestInitiatization(t *testing.T) {
	var e Endorser
	var err error
	e, err = NewEndorser(db.NewInMemDatabase())
	if e.(*endorser) != nil || err == nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", e, err)
	}
}
