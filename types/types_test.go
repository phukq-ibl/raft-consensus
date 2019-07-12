package types

import "testing"

func TestToString(t *testing.T) {
	ae := NewRequestVoteMsg()
	ae.Term = 1
	expect := `{"Term":1}`
	if expect != ae.ToString() {
		t.Errorf("Expect %s, real %s", expect, ae.ToString())
	}
}
