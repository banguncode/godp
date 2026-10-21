package dp

import "fmt"

// FMDFormat represents the fingerprint minutiae template format.
type FMDFormat int

const (
	// FMDFormatANSI378_2004 corresponds to ANSI INCITS 378-2004 standard format.
	FMDFormatANSI378_2004 FMDFormat = 0x001B0001
	// FMDFormatISO19794_2_2005 corresponds to ISO/IEC 19794-2:2005 standard format (Recommended).
	FMDFormatISO19794_2_2005 FMDFormat = 0x01010001
	// FMDFormatDPPreRegFeatures represents legacy DigitalPersona pre-registration features.
	FMDFormatDPPreRegFeatures FMDFormat = 0
	// FMDFormatDPRegFeatures represents legacy DigitalPersona registration template format.
	FMDFormatDPRegFeatures FMDFormat = 1
)

func (f FMDFormat) String() string {
	switch f {
	case FMDFormatANSI378_2004:
		return "ANSI_378_2004"
	case FMDFormatISO19794_2_2005:
		return "ISO_19794_2_2005"
	case FMDFormatDPPreRegFeatures:
		return "DP_PRE_REG_FEATURES"
	case FMDFormatDPRegFeatures:
		return "DP_REG_FEATURES"
	default:
		return fmt.Sprintf("UNKNOWN_FORMAT(0x%X)", int(f))
	}
}

// ParseFMDFormat parses string representations into FMDFormat.
func ParseFMDFormat(s string) (FMDFormat, error) {
	switch s {
	case "ISO_19794_2_2005", "iso", "ISO":
		return FMDFormatISO19794_2_2005, nil
	case "ANSI_378_2004", "ansi", "ANSI":
		return FMDFormatANSI378_2004, nil
	case "DP_PRE_REG_FEATURES":
		return FMDFormatDPPreRegFeatures, nil
	case "DP_REG_FEATURES":
		return FMDFormatDPRegFeatures, nil
	case "":
		return FMDFormatISO19794_2_2005, nil // Default
	default:
		return 0, fmt.Errorf("unsupported FMD format: %q", s)
	}
}

// FIDFormat represents the fingerprint image data format.
type FIDFormat int

const (
	// FIDFormatANSI381_2004 corresponds to ANSI INCITS 381-2004 format.
	FIDFormatANSI381_2004 FIDFormat = 0x001B0401
	// FIDFormatISO19794_4_2005 corresponds to ISO/IEC 19794-4:2005 format.
	FIDFormatISO19794_4_2005 FIDFormat = 0x01010007
)

func (f FIDFormat) String() string {
	switch f {
	case FIDFormatANSI381_2004:
		return "ANSI_381_2004"
	case FIDFormatISO19794_4_2005:
		return "ISO_19794_4_2005"
	default:
		return fmt.Sprintf("UNKNOWN_FID(0x%X)", int(f))
	}
}

// ParseFIDFormat parses string representations into FIDFormat.
func ParseFIDFormat(s string) (FIDFormat, error) {
	switch s {
	case "ISO_19794_4_2005", "iso", "ISO":
		return FIDFormatISO19794_4_2005, nil
	case "ANSI_381_2004", "ansi", "ANSI":
		return FIDFormatANSI381_2004, nil
	case "":
		return FIDFormatISO19794_4_2005, nil // Default
	default:
		return 0, fmt.Errorf("unsupported FID format: %q", s)
	}
}

// FingerPosition defines NIST/ISO standard finger positions.
type FingerPosition int

const (
	FingerUnknown     FingerPosition = 0
	FingerRightThumb  FingerPosition = 1
	FingerRightIndex  FingerPosition = 2
	FingerRightMiddle FingerPosition = 3
	FingerRightRing   FingerPosition = 4
	FingerRightLittle FingerPosition = 5
	FingerLeftThumb   FingerPosition = 6
	FingerLeftIndex   FingerPosition = 7
	FingerLeftMiddle  FingerPosition = 8
	FingerLeftRing    FingerPosition = 9
	FingerLeftLittle  FingerPosition = 10
)

func (p FingerPosition) String() string {
	names := [...]string{
		"Unknown", "RightThumb", "RightIndex", "RightMiddle", "RightRing", "RightLittle",
		"LeftThumb", "LeftIndex", "LeftMiddle", "LeftRing", "LeftLittle",
	}
	if int(p) >= 0 && int(p) < len(names) {
		return names[p]
	}
	return fmt.Sprintf("InvalidPosition(%d)", p)
}

const (
	// DefaultDissimilarityThreshold is the default False Match Rate (FAR 1/100,000) threshold.
	// Any comparison score <= DefaultDissimilarityThreshold is considered a match.
	DefaultDissimilarityThreshold uint = 21474 // (0x7FFFFFFF * 0.0001)

	// MaxFMDSize is the maximum buffer size required for ANSI/ISO FMD templates.
	MaxFMDSize = 34 * 1024
)

// Candidate represents a candidate match returned by 1:N Identification.
type Candidate struct {
	Index uint `json:"index"` // Index within the candidates array
	Score uint `json:"score"` // Dissimilarity score (0 = perfect match)
}
