package cmdx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeBody_atSign(t *testing.T) {
	assert.Equal(t, "@@", EscapeBody("@"))
	assert.Equal(t, "hello @@ world", EscapeBody("hello @ world"))
	assert.Equal(t, "@@@@", EscapeBody("@@"))
}

func TestEscapeBody_dollarSign(t *testing.T) {
	assert.Equal(t, "$$", EscapeBody("$"))
	assert.Equal(t, "$$5", EscapeBody("$5"))
	assert.Equal(t, "$$PATH", EscapeBody("$PATH"))
}

func TestEscapeBody_both(t *testing.T) {
	assert.Equal(t, "@@user $$5", EscapeBody("@user $5"))
}

func TestEscapeBody_noSpecialChars(t *testing.T) {
	assert.Equal(t, "hello world", EscapeBody("hello world"))
}

func TestUnescapeBody_atSign(t *testing.T) {
	assert.Equal(t, "@", UnescapeBody("@@"))
	assert.Equal(t, "hello @ world", UnescapeBody("hello @@ world"))
	assert.Equal(t, "@@", UnescapeBody("@@@@"))
}

func TestUnescapeBody_dollarSign(t *testing.T) {
	assert.Equal(t, "$", UnescapeBody("$$"))
	assert.Equal(t, "$5", UnescapeBody("$$5"))
}

func TestUnescapeBody_roundTrip(t *testing.T) {
	cases := []string{
		"hello @ world $ test",
		"@@ double at",
		"$$ double dollar",
		"no special chars",
		"",
		"@$@$",
	}
	for _, c := range cases {
		assert.Equal(t, c, UnescapeBody(EscapeBody(c)), "round-trip failed for %q", c)
	}
}

func TestEscapeDisplay(t *testing.T) {
	assert.Equal(t, `hello \> world`, EscapeDisplay("hello > world"))
	assert.Equal(t, "no arrow", EscapeDisplay("no arrow"))
}

func TestUnescapeDisplay(t *testing.T) {
	assert.Equal(t, "hello > world", UnescapeDisplay(`hello \> world`))
}

func TestEscapeDisplay_roundTrip(t *testing.T) {
	assert.Equal(t, "a > b > c", UnescapeDisplay(EscapeDisplay("a > b > c")))
}

func TestEscapeMeta(t *testing.T) {
	assert.Equal(t, `val\;ue`, EscapeMeta("val;ue"))
}

func TestUnescapeMeta(t *testing.T) {
	assert.Equal(t, "val;ue", UnescapeMeta(`val\;ue`))
}

func TestEscapeCell(t *testing.T) {
	assert.Equal(t, `val\|ue`, EscapeCell("val|ue"))
}

func TestUnescapeCell(t *testing.T) {
	assert.Equal(t, "val|ue", UnescapeCell(`val\|ue`))
}

func TestEscapeDesc(t *testing.T) {
	assert.Equal(t, `val\~ue`, EscapeDesc("val~ue"))
}

func TestUnescapeDesc(t *testing.T) {
	assert.Equal(t, "val~ue", UnescapeDesc(`val\~ue`))
}

func TestEscapeCodeLine(t *testing.T) {
	assert.Equal(t, `\@SomeAnnotation`, EscapeCodeLine("@SomeAnnotation"))
	assert.Equal(t, `\@/CODE`, EscapeCodeLine("@/CODE"))
	assert.Equal(t, "func main() {}", EscapeCodeLine("func main() {}"))
	assert.Equal(t, "  @indented", EscapeCodeLine("  @indented")) // @ not at start
}

func TestUnescapeCodeLine(t *testing.T) {
	assert.Equal(t, "@SomeAnnotation", UnescapeCodeLine(`\@SomeAnnotation`))
	assert.Equal(t, "@/CODE", UnescapeCodeLine(`\@/CODE`))
	assert.Equal(t, "func main() {}", UnescapeCodeLine("func main() {}"))
}

func TestEscapeCodeLine_roundTrip(t *testing.T) {
	cases := []string{
		"@SomeAnnotation",
		"@/CODE",
		"func main() {}",
		"  @indented",
		"",
		"normal line",
	}
	for _, c := range cases {
		assert.Equal(t, c, UnescapeCodeLine(EscapeCodeLine(c)), "round-trip failed for %q", c)
	}
}
