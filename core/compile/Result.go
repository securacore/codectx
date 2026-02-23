package compile

// Result holds the outcome of a compile operation.
type Result struct {
	OutputDir   string
	FilesCopied int
	Packages    int
}
