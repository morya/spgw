package ptr

import (
	"testing"
)

func TestBool(t *testing.T) {
	v1 := MakeBool(true)
	v2 := MakeBool(true)
	t.Logf("v = %v", *v1)
	t.Logf("v = %p %p", v1, v2)
}
