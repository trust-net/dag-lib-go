package stack

import (
    "testing"
	"github.com/trust-net/dag-lib-go/db"
)

func TestInitiatization(t *testing.T) {
	var stack DLT
	var err error
	stack, err = NewDltStack(db.NewInMemDatabase())
	if stack.(*dlt) != nil || err == nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", stack, err)
	}
}
