package manifest

// Manifest represents a package.yml data map.
// Every documentation package (local, installed, compiled) uses this format.
type Manifest struct {
	Name        string            `yaml:"name"`
	Author      string            `yaml:"author"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Foundation  []FoundationEntry `yaml:"foundation,omitempty"`
	Topics      []TopicEntry      `yaml:"topics,omitempty"`
	Prompts     []PromptEntry     `yaml:"prompts,omitempty"`
	Plans       []PlanEntry       `yaml:"plans,omitempty"`
}
