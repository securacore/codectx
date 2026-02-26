# Edge Cases

## Special Characters

Text with @ sign and $ sign and > arrow.

Text with @@ double at and $$ double dollar.

## Code Block Edge Cases

Code with @ at line start:

```go
@SomeAnnotation
func main() {}
@/CODE
fmt.Println("not a closing tag")
```

## Nested Formatting

**Bold with *italic* inside** and *italic with **bold** inside*.

***Bold italic text***

A [link with > arrow](https://example.com/path?a>b) in text.

## Tables With No Pattern

| Animal | Sound |
|--------|-------|
| Cat    | Meow  |
| Dog    | Woof  |

## Empty Sections

---

## Dollar Signs

The cost is $5 per item. Variable $PATH is set. Use $$double for escaping.
