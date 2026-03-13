package version

import (
	"testing"
)

func TestBumpVersion_Patch(t *testing.T) {
	got, err := bumpVersion("1.2.3", "patch")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.2.4" {
		t.Errorf("bumpVersion(\"1.2.3\", \"patch\") = %q, want %q", got, "1.2.4")
	}
}

func TestBumpVersion_Minor(t *testing.T) {
	got, err := bumpVersion("1.2.3", "minor")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.3.0" {
		t.Errorf("bumpVersion(\"1.2.3\", \"minor\") = %q, want %q", got, "1.3.0")
	}
}

func TestBumpVersion_Major(t *testing.T) {
	got, err := bumpVersion("1.2.3", "major")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.0.0" {
		t.Errorf("bumpVersion(\"1.2.3\", \"major\") = %q, want %q", got, "2.0.0")
	}
}

func TestBumpVersion_ZeroBase(t *testing.T) {
	got, err := bumpVersion("0.1.0", "patch")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.1.1" {
		t.Errorf("bumpVersion(\"0.1.0\", \"patch\") = %q, want %q", got, "0.1.1")
	}
}

func TestBumpVersion_MajorFromZero(t *testing.T) {
	got, err := bumpVersion("0.5.3", "major")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.0.0" {
		t.Errorf("bumpVersion(\"0.5.3\", \"major\") = %q, want %q", got, "1.0.0")
	}
}

func TestBumpVersion_StripsVPrefix(t *testing.T) {
	got, err := bumpVersion("v1.2.3", "patch")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.2.4" {
		t.Errorf("bumpVersion(\"v1.2.3\", \"patch\") = %q, want %q", got, "1.2.4")
	}
}

func TestBumpVersion_InvalidFormat(t *testing.T) {
	_, err := bumpVersion("1.2", "patch")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestBumpVersion_InvalidMajor(t *testing.T) {
	_, err := bumpVersion("x.2.3", "patch")
	if err == nil {
		t.Error("expected error for invalid major")
	}
}

func TestBumpVersion_InvalidMinor(t *testing.T) {
	_, err := bumpVersion("1.x.3", "patch")
	if err == nil {
		t.Error("expected error for invalid minor")
	}
}

func TestBumpVersion_InvalidPatch(t *testing.T) {
	_, err := bumpVersion("1.2.x", "patch")
	if err == nil {
		t.Error("expected error for invalid patch")
	}
}
