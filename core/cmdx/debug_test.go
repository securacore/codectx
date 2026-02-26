package cmdx

import (
	"testing"
)

func TestDebugTilde3(t *testing.T) {
	input := []byte("~~0~0~")

	a, _ := DumpAST(input)
	t.Logf("AST A:\n%s", a)

	encoded, err := Encode(input)
	if err != nil {
		t.Fatalf("encode err: %v", err)
	}
	t.Logf("ENCODED:\n%q", string(encoded))

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	t.Logf("DECODED:\n%q", string(decoded))

	b, _ := DumpAST(decoded)
	t.Logf("AST B:\n%s", b)
}
