package dp

import (
	"testing"
)

func TestParseFMDFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected FMDFormat
		wantErr  bool
	}{
		{"ISO_19794_2_2005", FMDFormatISO19794_2_2005, false},
		{"ANSI_378_2004", FMDFormatANSI378_2004, false},
		{"iso", FMDFormatISO19794_2_2005, false},
		{"", FMDFormatISO19794_2_2005, false}, // Default
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseFMDFormat(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseFMDFormat(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParseFMDFormat(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEnrollmentValidation(t *testing.T) {
	// Must fail with less than 2 samples
	_, err := CreateEnrollmentFMD([][]byte{{1, 2, 3}}, FMDFormatISO19794_2_2005)
	if err == nil {
		t.Error("expected error for less than 2 samples, got nil")
	}
}

func TestCompareValidation(t *testing.T) {
	// Empty buffers must fail
	_, _, err := Compare(nil, nil, FMDFormatISO19794_2_2005, 0)
	if err == nil {
		t.Error("expected error for empty buffers, got nil")
	}
}
