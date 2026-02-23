package errs

func New(kind Kind, message string) *Error {
	return &Error{
		Kind:    kind,
		Message: message,
	}
}

func Wrap(kind Kind, message string, err error) *Error {
	return &Error{
		Kind:    kind,
		Message: message,
		Err:     err,
	}
}
