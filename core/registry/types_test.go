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
			want:  DepKey{Name: "react-patterns", Org: "community", Version: "latest"},
		},
		{
			name:  "exact version",
			input: "company-standards@acme:2.0.0",
			want:  DepKey{Name: "company-standards", Org: "acme", Version: "2.0.0"},
		},
		{
			name:  "complex version",
			input: "tailwind-guide@designteam:2.1.0",
			want:  DepKey{Name: "tailwind-guide", Org: "designteam", Version: "2.1.0"},
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
			name:    "empty org",
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

	dk := DepKey{Name: "react-patterns", Org: "community", Version: "latest"}
	if got := dk.String(); got != "react-patterns@community:latest" {
		t.Errorf("String() = %q, want %q", got, "react-patterns@community:latest")
	}
}

func TestDepKeyPackageRef(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Org: "community", Version: "2.3.1"}
	if got := dk.PackageRef(); got != "react-patterns@community" {
		t.Errorf("PackageRef() = %q, want %q", got, "react-patterns@community")
	}
}

func TestDepKeyRepoName(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Org: "community", Version: "latest"}
	if got := dk.RepoName(); got != "codectx-react-patterns" {
		t.Errorf("RepoName() = %q, want %q", got, "codectx-react-patterns")
	}
}

func TestDepKeyRepoURL(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Org: "community", Version: "latest"}
	got := dk.RepoURL("github.com")
	want := "https://github.com/community/codectx-react-patterns"
	if got != want {
		t.Errorf("RepoURL() = %q, want %q", got, want)
	}
}

func TestDepKeyRepoPath(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Org: "community", Version: "latest"}
	if got := dk.RepoPath(); got != "community/codectx-react-patterns" {
		t.Errorf("RepoPath() = %q, want %q", got, "community/codectx-react-patterns")
	}
}

func TestDepKeyDirName(t *testing.T) {
	t.Parallel()

	dk := DepKey{Name: "react-patterns", Org: "community", Version: "2.3.1"}
	if got := dk.DirName(); got != "react-patterns@community:2.3.1" {
		t.Errorf("DirName() = %q, want %q", got, "react-patterns@community:2.3.1")
	}
}

func TestParsePackageRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantName string
		wantOrg  string
		wantErr  bool
	}{
		{"valid", "react-patterns@community", "react-patterns", "community", false},
		{"missing at", "react-patterns", "", "", true},
		{"empty name", "@community", "", "", true},
		{"empty org", "react-patterns@", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name, org, err := ParsePackageRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tt.wantName || org != tt.wantOrg {
				t.Errorf("got (%q, %q), want (%q, %q)", name, org, tt.wantName, tt.wantOrg)
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
