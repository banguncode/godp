package dto

import "encoding/json"

// ExtractFMDRequest defines the payload for extracting an FMD from raw grayscale image bytes.
type ExtractFMDRequest struct {
	RawImageBase64 string `json:"raw_image_base64"`
	Width          uint   `json:"width"`
	Height         uint   `json:"height"`
	DPI            uint   `json:"dpi"`
	FingerPosition int    `json:"finger_position"`
	Format         string `json:"format"`
}

// EnrollRequest defines the payload for creating a master enrollment FMD from multiple candidate samples.
type EnrollRequest struct {
	SessionID string   `json:"session_id,omitempty"`
	Format    string   `json:"format"` // ignored for enrollment: output is always DP_REG_FEATURES
	// Samples are RAW captures of the SAME finger, each either a DigitalPersona WebSDK capture
	// ({"Data":"<base64url>","Format":{"iWidth":..,"iHeight":..,"iXdpi":..}}, as an object or a JSON string)
	// or a plain base64 string of 8bpp pixels (then Width/Height/DPI below are required).
	// PNG/JPEG and pre-extracted FMDs are rejected.
	Samples        []json.RawMessage `json:"samples"`
	Width          uint              `json:"width,omitempty"`
	Height         uint              `json:"height,omitempty"`
	DPI            uint              `json:"dpi,omitempty"`
	FingerPosition int               `json:"finger_position,omitempty"`
}

// CompareRequest defines the payload for 1:1 fingerprint verification.
type CompareRequest struct {
	ProbeFMDBase64    string `json:"probe_fmd_base64"`
	EnrolledFMDBase64 string `json:"enrolled_fmd_base64"`
	Format            string `json:"format"`
	Threshold         uint   `json:"threshold,omitempty"` // Dissimilarity threshold (default: 21474)
}

// IdentifyRequest defines the payload for 1:N biometric search.
type IdentifyRequest struct {
	ProbeFMDBase64      string   `json:"probe_fmd_base64"`
	CandidateFMDsBase64 []string `json:"candidate_fmds_base64"`
	Format              string   `json:"format"`
	Threshold           uint     `json:"threshold,omitempty"`
}

// ConvertFMDRequest defines the payload for converting FMD formats.
type ConvertFMDRequest struct {
	FMDBase64  string `json:"fmd_base64"`
	FromFormat string `json:"from_format"`
	ToFormat   string `json:"to_format"`
}

// NFIQRequest defines the payload for NIST NFIQ fingerprint quality assessment.
type NFIQRequest struct {
	RawImageBase64 string `json:"raw_image_base64"`
	Width          uint   `json:"width"`
	Height         uint   `json:"height"`
	DPI            uint   `json:"dpi"`
}
