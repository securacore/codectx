package md

import (
	"testing"
)

func TestDebugTilde3(t *testing.T) {
	input := []byte("~~0~0~")

	a, _ := dumpAST(input)
	t.Logf("AST A:\n%s", a)

	encoded, err := Encode(input)
	if err != nil {
		t.Fatalf("encode err: %v", err)
	}
	t.Logf("ENCODED:\n%q", string(encoded))

	b, _ := dumpAST(encoded)
	t.Logf("AST B:\n%s", b)
}
