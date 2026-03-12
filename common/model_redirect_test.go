package common

import "testing"

func TestGetSystemRedirectedModelName_MolePrefix(t *testing.T) {
	redirected, ok := GetSystemRedirectedModelName("mole-gpt-4o")
	if !ok {
		t.Fatalf("expected redirected=true")
	}
	if redirected != "gpt-4o" {
		t.Fatalf("expected redirected model gpt-4o, got %s", redirected)
	}
}

func TestGetSystemRedirectedModelName_NonMolePrefixNotRedirected(t *testing.T) {
	redirected, ok := GetSystemRedirectedModelName("moleapi-gpt-4o")
	if ok {
		t.Fatalf("expected redirected=false")
	}
	if redirected != "moleapi-gpt-4o" {
		t.Fatalf("expected model to remain moleapi-gpt-4o, got %s", redirected)
	}
}

func TestGetSystemRedirectedModelName_IgnoreEmptyTarget(t *testing.T) {
	redirected, ok := GetSystemRedirectedModelName("mole-")
	if ok {
		t.Fatalf("expected redirected=false")
	}
	if redirected != "mole-" {
		t.Fatalf("expected model to remain mole-, got %s", redirected)
	}
}
