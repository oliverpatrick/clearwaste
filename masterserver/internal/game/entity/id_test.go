package entity

import "testing"

func TestZeroIsReservedInvalid(t *testing.T) {
	if Invalid != ID(0) {
		t.Fatalf("Invalid=%d", Invalid)
	}
	if Invalid.Valid() {
		t.Fatal("zero entity ID is valid")
	}
	if !ID(1).Valid() {
		t.Fatal("non-zero entity ID is invalid")
	}
}
