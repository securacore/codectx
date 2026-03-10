# Code Examples

## Go Function

```go
func CreateUser(name string, email string) (*User, error) {
    user := &User{
        Name:  name,
        Email: email,
    }
    if err := db.Create(user).Error; err != nil {
        return nil, fmt.Errorf("creating user: %w", err)
    }
    return user, nil
}
```

## Python Script

```python
def process_data(items: list[dict]) -> list[dict]:
    """Process and validate input data."""
    results = []
    for item in items:
        if validate(item):
            results.append(transform(item))
    return results
```

## Shell Commands

```bash
# Build and test
go build ./...
go test -v ./...

# Run with environment
DATABASE_URL=postgres://localhost/mydb go run .
```

## Unlabeled Code

```
just plain text in a code block
no language specified
```

## Indented Code

    this is an indented code block
    it has no language tag
    and uses 4-space indentation
