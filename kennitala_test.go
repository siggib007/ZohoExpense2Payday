package main

import (
	"testing"

	"github.com/siggib007/goutils/kennitala"
)

func TestValidateKT(t *testing.T) {
	// Valid — fake but mathematically correct, Jan 01 2010
	lstValid := []string{
		"0101103190",  // no separator
		"010110-3190", // with dash
		"010110 3190", // with space
	}
	for _, strKT := range lstValid {
		if !kennitala.ValidateKT(strKT) {
			t.Errorf("expected %s to be valid but got invalid", strKT)
		}
	}

	// Invalid
	lstInvalid := []string{
		"",
		"1234",
		"abcdefghij", // not digits
		"0101103180", // wrong check digit
		"0101103170", // wrong check digit
		"9901103190", // impossible date, check digit mismatch
		"0101103100", // wrong check digit
	}
	for _, strKT := range lstInvalid {
		if kennitala.ValidateKT(strKT) {
			t.Errorf("expected %s to be invalid but got valid", strKT)
		}
	}
}
