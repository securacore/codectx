package plan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/securacore/codectx/core/project"
	"gopkg.in/yaml.v3"
)

// planHeader is the comment header written at the top of plan.yml files.
const planHeader = "# codectx plan state\n# Tracks multi-step workflow with dependency hashes for drift detection.\n# Checked into version control for cross-machine continuity.\n\n"

// Load reads and parses a plan.yml file from the given path.
func Load(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading plan: %w", err)
	}

	var p Plan
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing plan: %w", err)
	}

	// Ensure slices are non-nil for safe iteration.
	if p.Dependencies == nil {
		p.Dependencies = []Dependency{}
	}
	if p.Steps == nil {
		p.Steps = []Step{}
	}
	for i := range p.Steps {
		if p.Steps[i].Queries == nil {
			p.Steps[i].Queries = []string{}
		}
		if p.Steps[i].Chunks == nil {
			p.Steps[i].Chunks = []string{}
		}
		if p.Steps[i].BlockedBy == nil {
			p.Steps[i].BlockedBy = []int{}
		}
	}

	return &p, nil
}

// Save writes a plan to the given path as YAML with 2-space indentation.
func Save(path string, p *Plan) error {
	return project.WriteYAMLFile(path, planHeader, p)
}

// Discover finds plan directories within the given documentation root.
// It looks for directories under docs/plans/ that contain a plan.yml file.
// Returns a map of plan name (directory name) to the absolute path of plan.yml.
func Discover(rootDir string) (map[string]string, error) {
	plansDir := filepath.Join(rootDir, "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading plans directory: %w", err)
	}

	plans := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		planPath := filepath.Join(plansDir, entry.Name(), PlanFile)
		if _, statErr := os.Stat(planPath); statErr == nil {
			plans[entry.Name()] = planPath
		}
	}

	return plans, nil
}

// FindPlan locates a plan by name within the documentation root.
// If planName is empty and exactly one plan exists, it returns that plan.
// Returns the plan name and the absolute path to its plan.yml.
func FindPlan(rootDir, planName string) (string, string, error) {
	plans, err := Discover(rootDir)
	if err != nil {
		return "", "", err
	}

	if len(plans) == 0 {
		return "", "", fmt.Errorf("no plans found in %s/plans/", rootDir)
	}

	if planName != "" {
		path, ok := plans[planName]
		if !ok {
			available := make([]string, 0, len(plans))
			for name := range plans {
				available = append(available, name)
			}
			return "", "", fmt.Errorf("plan %q not found; available: %v", planName, available)
		}
		return planName, path, nil
	}

	// No name specified — require exactly one plan.
	if len(plans) == 1 {
		for name, path := range plans {
			return name, path, nil
		}
	}

	available := make([]string, 0, len(plans))
	for name := range plans {
		available = append(available, name)
	}
	return "", "", fmt.Errorf("multiple plans found, specify a name: %v", available)
}
