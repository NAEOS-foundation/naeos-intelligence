package main

import "testing"

func TestApplicationNameIsDefined(t *testing.T) {
	if applicationName == "" {
		t.Fatal("applicationName must not be empty")
	}
}
