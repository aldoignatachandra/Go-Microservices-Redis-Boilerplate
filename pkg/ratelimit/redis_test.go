package ratelimit

import "testing"

func TestBuildKeyPrefix(t *testing.T) {
	t.Parallel()

	got := BuildKeyPrefix("user-local", "service-user")
	want := "ratelimit:user-local:service-user"

	if got != want {
		t.Fatalf("BuildKeyPrefix() = %q, want %q", got, want)
	}
}

func TestBuildKeyPrefix_DefaultValues(t *testing.T) {
	t.Parallel()

	got := BuildKeyPrefix("  ", "")
	want := "ratelimit:default:service"

	if got != want {
		t.Fatalf("BuildKeyPrefix() = %q, want %q", got, want)
	}
}
