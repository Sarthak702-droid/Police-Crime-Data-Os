package utils

import (
	"bytes"
	"testing"
)

func TestBuildTextPDF(t *testing.T) {
	pdf := BuildTextPDF("Test Export", []string{"Officer: KG123", "A line with (parentheses) and \\ slash"})
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatal("missing PDF header")
	}
	if !bytes.Contains(pdf, []byte("startxref")) || !bytes.HasSuffix(pdf, []byte("%%EOF\n")) {
		t.Fatal("invalid PDF trailer")
	}
	if len(pdf) < 500 {
		t.Fatalf("unexpectedly small PDF: %d", len(pdf))
	}
}
