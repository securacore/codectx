package registry

import (
	"testing"
)

func TestParseDepKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    DepKey
		wantErr bool
	}{
		{
			name:  "standard key",
			input: "react-patterns@community:latest",
			want:  DepKey{Name: "react-patterns", Author: "community", Version: "latest"},
		},
		{
			name:  "exact version",
			input: "company-standards@acme:2.0.0",
			want:  DepKey{Name: "company-standards", Author: "acme", Version: "2.0.0"},
		},
		{
			name:  "complex version",
			input: "tailwind-guide@designteam:2.1.0",
			want:  DepKey{Name: "tailwind-guide", Author: "designteam", Version: "2.1.0"},
		},
		{
			name:    "missing at sign",
			input:   "react-patterns:latest",
			wantErr: true,
		},
		{
			name:    "missing colon",
			input:   "react-patterns@community",
			wantErr: true,
		},
		{
			name:    "empty name",
			input:   "@community:latest",
			wantErr: true,
		},
		{
			name:    "empty author",
			input:   "react-patterns@:latest",
			wantErr: true,
		},
		{
			name:    "empty version",
			input:   "react-patterns@community:",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDepKey(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got %+v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseDepKey(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDepKeyString(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Author: "community", Version: "latest"}
	if got := dk.String(); got != "react-patterns@community:latest" {
		t.Errorf("String() = %q, want %q", got, "react-patterns@community:latest")
	}
}

func TestDepKeyPackageRef(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Author: "community", Version: "2.3.1"}
	if got := dk.PackageRef(); got != "react-patterns@community" {
		t.Errorf("PackageRef() = %q, want %q", got, "react-patterns@community")
	}
}

func TestDepKeyRepoName(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Author: "community", Version: "latest"}
	if got := dk.RepoName(); got != "codectx-react-patterns" {
		t.Errorf("RepoName() = %q, want %q", got, "codectx-react-patterns")
	}
}

func TestDepKeyRepoURL(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Author: "community", Version: "latest"}
	got := dk.RepoURL("github.com")
	want := "https://github.com/community/codectx-react-patterns"
	if got != want {
		t.Errorf("RepoURL() = %q, want %q", got, want)
	}
}

func TestParsePackageRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantName   string
		wantAuthor string
		wantErr    bool
	}{
		{"valid", "react-patterns@community", "react-patterns", "community", false},
		{"missing at", "react-patterns", "", "", true},
		{"empty name", "@community", "", "", true},
		{"empty author", "react-patterns@", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name, author, err := ParsePackageRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tt.wantName || author != tt.wantAuthor {
				t.Errorf("got (%q, %q), want (%q, %q)", name, author, tt.wantName, tt.wantAuthor)
			}
		})
	}
}

func TestGitTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"2.3.1", "v2.3.1"},
		{"v2.3.1", "v2.3.1"},
		{"0.1.0", "v0.1.0"},
	}

	for _, tt := range tests {
		if got := GitTag(tt.input); got != tt.want {
			t.Errorf("GitTag(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestVersionFromTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"v2.3.1", "2.3.1"},
		{"2.3.1", "2.3.1"},
		{"v0.1.0", "0.1.0"},
	}

	for _, tt := range tests {
		if got := VersionFromTag(tt.input); got != tt.want {
			t.Errorf("VersionFromTag(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParsePartialDepKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    PartialDepKey
		wantErr bool
	}{
		{
			name:  "name only",
			input: "react",
			want:  PartialDepKey{Name: "react"},
		},
		{
			name:  "name@author",
			input: "react@community",
			want:  PartialDepKey{Name: "react", Author: "community"},
		},
		{
			name:  "name@author:version",
			input: "react@community:2.0.0",
			want:  PartialDepKey{Name: "react", Author: "community", Version: "2.0.0"},
		},
		{
			name:  "name:version",
			input: "react:2.0.0",
			want:  PartialDepKey{Name: "react", Version: "2.0.0"},
		},
		{
			name:  "name with hyphens",
			input: "react-patterns@community",
			want:  PartialDepKey{Name: "react-patterns", Author: "community"},
		},
		{
			name:  "latest version",
			input: "react@community:latest",
			want:  PartialDepKey{Name: "react", Author: "community", Version: "latest"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "empty name with @",
			input:   "@community",
			wantErr: true,
		},
		{
			name:    "empty name with :",
			input:   ":2.0.0",
			wantErr: true,
		},
		{
			name:  "whitespace trimmed",
			input: "  react  ",
			want:  PartialDepKey{Name: "react"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParsePartialDepKey(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParsePartialDepKey(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPartialDepKey_IsComplete(t *testing.T) {
	t.Parallel()

	if (PartialDepKey{Name: "a", Author: "b", Version: "1.0"}).IsComplete() != true {
		t.Error("full key should be complete")
	}
	if (PartialDepKey{Name: "a", Author: "b"}).IsComplete() != false {
		t.Error("missing version should not be complete")
	}
	if (PartialDepKey{Name: "a"}).IsComplete() != false {
		t.Error("name-only should not be complete")
	}
}

func TestPartialDepKey_ToDepKey(t *testing.T) {
	t.Parallel()

	// With version.
	dk, err := (PartialDepKey{Name: "react", Author: "community", Version: "2.0.0"}).ToDepKey()
	if err != nil {
		t.Fatal(err)
	}
	if dk.Name != "react" || dk.Author != "community" || dk.Version != "2.0.0" {
		t.Errorf("unexpected DepKey: %+v", dk)
	}

	// Without version — defaults to "latest".
	dk, err = (PartialDepKey{Name: "react", Author: "community"}).ToDepKey()
	if err != nil {
		t.Fatal(err)
	}
	if dk.Version != "latest" {
		t.Errorf("expected version 'latest', got %q", dk.Version)
	}

	// Without author — error.
	_, err = (PartialDepKey{Name: "react"}).ToDepKey()
	if err == nil {
		t.Error("expected error for missing author")
	}
}
