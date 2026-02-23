package errs

type Kind int

const (
	KindInternal   Kind = iota // unexpected internal error
	KindNotFound               // resource not found
	KindInvalid                // invalid input or state
	KindPermission             // insufficient permissions
	KindConflict               // conflicting state
	KindExists                 // resource already exists
)

func (k Kind) String() string {
	switch k {
	case KindNotFound:
		return "not found"
	case KindInvalid:
		return "invalid"
	case KindPermission:
		return "permission denied"
	case KindConflict:
		return "conflict"
	case KindExists:
		return "already exists"
	default:
		return "internal error"
	}
}
