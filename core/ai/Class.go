package ai

// ModelClass describes a documentation compatibility target. The class
// defines the minimum model capability tier that compiled documentation
// is written for. It is NOT the model being used — it controls how
// documentation is authored for cross-model consumption.
type ModelClass struct {
	// ID is the configuration identifier (e.g., "gpt-4o-class").
	ID string

	// Name is the human-readable display name (e.g., "GPT-4o class").
	Name string

	// Description summarizes the capability tier this class represents.
	Description string
}

// Classes is the default registry of known model classes.
var Classes = []ModelClass{
	{
		ID:          "gpt-4o-class",
		Name:        "GPT-4o class",
		Description: "Mid-tier instruction-following models (GPT-4o, Claude Sonnet, Gemini Pro)",
	},
	{
		ID:          "claude-sonnet-class",
		Name:        "Claude Sonnet class",
		Description: "Strong reasoning models with extended context (Claude Sonnet, GPT-4o)",
	},
	{
		ID:          "o1-class",
		Name:        "o1 class",
		Description: "Frontier reasoning models (o1, Claude Opus, Gemini Ultra)",
	},
}

// ClassByID returns the model class with the given ID and true,
// or a zero ModelClass and false if no match is found.
func ClassByID(id string) (ModelClass, bool) {
	for _, c := range Classes {
		if c.ID == id {
			return c, true
		}
	}
	return ModelClass{}, false
}
