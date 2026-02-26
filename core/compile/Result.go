package compile

// Result holds the outcome of a compile operation.
type Result struct {
	OutputDir     string
	ObjectsStored int
	ObjectsPruned int
	Packages      int
	Dedup         DeduplicationReport
	UpToDate      bool // true if inputs unchanged since last compile
	Compressed    bool // true if CMDX compression was used
	Heuristics    *Heuristics
	Entries       []ResultEntry // per-file listing for display
}

// ResultEntry describes a single compiled object for display.
type ResultEntry struct {
	Section string // "foundation", "application", "topics", "prompts", "plans"
	ID      string
	Object  string // object path (e.g., "objects/a1b2c3d4.md")
	Size    int    // bytes on disk
}

// ProgressEvent is emitted during compilation to report progress.
type ProgressEvent struct {
	Stage   string // "prepare", "merge", "store", "finalize"
	Message string
}
