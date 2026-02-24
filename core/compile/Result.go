package compile

// Result holds the outcome of a compile operation.
type Result struct {
	OutputDir     string
	ObjectsStored int
	ObjectsPruned int
	Packages      int
	Dedup         DeduplicationReport
	UpToDate      bool // true if inputs unchanged since last compile
}
