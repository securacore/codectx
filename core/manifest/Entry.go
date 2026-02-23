package manifest

// FoundationEntry represents a foundational document in the data map.
type FoundationEntry struct {
	ID          string   `yaml:"id"`
	Path        string   `yaml:"path"`
	Load        string   `yaml:"load,omitempty"`
	Description string   `yaml:"description"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	RequiredBy  []string `yaml:"required_by,omitempty"`
}

// TopicEntry represents a topic document in the data map.
type TopicEntry struct {
	ID          string   `yaml:"id"`
	Path        string   `yaml:"path"`
	Description string   `yaml:"description"`
	Spec        string   `yaml:"spec,omitempty"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	RequiredBy  []string `yaml:"required_by,omitempty"`
	Files       []string `yaml:"files,omitempty"`
}

// PromptEntry represents a prompt definition in the data map.
type PromptEntry struct {
	ID          string   `yaml:"id"`
	Path        string   `yaml:"path"`
	Description string   `yaml:"description"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	RequiredBy  []string `yaml:"required_by,omitempty"`
}

// PlanEntry represents an implementation plan in the data map.
type PlanEntry struct {
	ID          string   `yaml:"id"`
	Path        string   `yaml:"path"`
	State       string   `yaml:"state"`
	Description string   `yaml:"description"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	RequiredBy  []string `yaml:"required_by,omitempty"`
}
