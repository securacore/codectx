package manifest_test

import (
	"testing"

	"github.com/securacore/codectx/core/manifest"
)

func TestClassifyDocType_Foundation(t *testing.T) {
	got := manifest.ClassifyDocType("foundation/coding-standards/README.md")
	if got != manifest.DocFoundation {
		t.Errorf("expected %q, got %q", manifest.DocFoundation, got)
	}
}

func TestClassifyDocType_Topic(t *testing.T) {
	got := manifest.ClassifyDocType("topics/authentication/jwt-tokens.md")
	if got != manifest.DocTopic {
		t.Errorf("expected %q, got %q", manifest.DocTopic, got)
	}
}

func TestClassifyDocType_Plan(t *testing.T) {
	got := manifest.ClassifyDocType("plans/auth-migration/README.md")
	if got != manifest.DocPlan {
		t.Errorf("expected %q, got %q", manifest.DocPlan, got)
	}
}

func TestClassifyDocType_Prompt(t *testing.T) {
	got := manifest.ClassifyDocType("prompts/code-review/README.md")
	if got != manifest.DocPrompt {
		t.Errorf("expected %q, got %q", manifest.DocPrompt, got)
	}
}

func TestClassifyDocType_System(t *testing.T) {
	got := manifest.ClassifyDocType("system/topics/taxonomy-generation/README.md")
	if got != manifest.DocSystem {
		t.Errorf("expected %q, got %q", manifest.DocSystem, got)
	}
}

func TestClassifyDocType_Package(t *testing.T) {
	got := manifest.ClassifyDocType(".codectx/packages/react-patterns@community:2.1.0/foundation/README.md")
	if got != manifest.DocPackage {
		t.Errorf("expected %q, got %q", manifest.DocPackage, got)
	}
}

func TestClassifyDocType_DefaultIsTopic(t *testing.T) {
	got := manifest.ClassifyDocType("some/unknown/path.md")
	if got != manifest.DocTopic {
		t.Errorf("expected default %q, got %q", manifest.DocTopic, got)
	}
}

func TestClassifyDocType_SpecFileStillClassifiedByPath(t *testing.T) {
	// Spec files are classified by directory path, not by suffix.
	// The ChunkType (spec vs object) is a separate concern from DocumentType.
	got := manifest.ClassifyDocType("topics/auth/jwt.spec.md")
	if got != manifest.DocTopic {
		t.Errorf("expected %q for spec under topics, got %q", manifest.DocTopic, got)
	}
}

func TestStringPtr(t *testing.T) {
	s := "test"
	ptr := manifest.StringPtr(s)
	if ptr == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *ptr != s {
		t.Errorf("expected %q, got %q", s, *ptr)
	}
}
