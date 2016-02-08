package main

import "testing"

func TestUnused(t *testing.T) {
	count := myloader([]string{"./testdata/"})
	if count != 2 {
		t.Fatal("Expected to find 2 unused var, but got %v", count)
	}
}
