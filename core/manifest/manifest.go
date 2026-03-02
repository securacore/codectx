package manifest

// Manifest represents a manifest.yml data map.
// Every documentation package (local, installed, compiled) uses this format.
type Manifest struct {
	Name        string             `yaml:"name"`
	Author      string             `yaml:"author"`
	Version     string             `yaml:"version"`
	Description string             `yaml:"description"`
	Foundation  []FoundationEntry  `yaml:"foundation,omitempty"`
	Application []ApplicationEntry `yaml:"application,omitempty"`
	Topics      []TopicEntry       `yaml:"topics,omitempty"`
	Prompts     []PromptEntry      `yaml:"prompts,omitempty"`
	Plans       []PlanEntry        `yaml:"plans,omitempty"`
}
