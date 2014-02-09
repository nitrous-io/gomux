package gomux

import (
	"testing"
)

func TestRandomString(t *testing.T) {
	s := RandomString()
	if len(s) != 64 {
		t.Fail()
	}
}
