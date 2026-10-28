package server

import "testing"

func TestIsAllowedOrigin(t *testing.T) {
	for origin, want := range map[string]bool{
		"":                            true,
		"https://live.warthunder.com": true,
		"moz-extension://abc-123":     true,
		"https://evil.example":        false,
		"https://live.warthunder.com.evil.example": false,
	} {
		if got := isAllowedOrigin(origin); got != want {
			t.Errorf("%q: got %v, want %v", origin, got, want)
		}
	}
}
