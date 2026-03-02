package ide

import "fmt"

// Phase represents a stage in the documentation authoring conversation.
type Phase int

const (
	PhaseDiscover Phase = iota // Understand the subject, audience, and purpose
	PhaseClassify              // Recommend a documentation category and ID
	PhaseScope                 // Define boundaries and dependencies
	PhaseDraft                 // Author document section by section
	PhaseReview                // Validate against codectx standards
	PhaseFinalize              // Present document in <document> tags for preview
	PhaseComplete              // Document approved and written to disk
)

// String returns the phase name for YAML serialization and display.
func (p Phase) String() string {
	switch p {
	case PhaseDiscover:
		return "discover"
	case PhaseClassify:
		return "classify"
	case PhaseScope:
		return "scope"
	case PhaseDraft:
		return "draft"
	case PhaseReview:
		return "review"
	case PhaseFinalize:
		return "finalize"
	case PhaseComplete:
		return "complete"
	default:
		return "unknown"
	}
}

// parsePhase converts a string to a Phase.
func parsePhase(s string) (Phase, error) {
	switch s {
	case "discover":
		return PhaseDiscover, nil
	case "classify":
		return PhaseClassify, nil
	case "scope":
		return PhaseScope, nil
	case "draft":
		return PhaseDraft, nil
	case "review":
		return PhaseReview, nil
	case "finalize":
		return PhaseFinalize, nil
	case "complete":
		return PhaseComplete, nil
	default:
		return 0, fmt.Errorf("unknown phase: %q", s)
	}
}

// MarshalYAML serializes the phase as a string.
func (p Phase) MarshalYAML() (any, error) {
	return p.String(), nil
}

// UnmarshalYAML deserializes a phase from a string.
func (p *Phase) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := parsePhase(s)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
