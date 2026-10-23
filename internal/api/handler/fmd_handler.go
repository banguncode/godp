package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"

	"godp/internal/api/dto"
	"godp/pkg/dp"
)

// FMDHandler processes HTTP requests related to fingerprint minutiae data.
type FMDHandler struct{}

func NewFMDHandler() *FMDHandler {
	return &FMDHandler{}
}

// isPNG checks if the byte slice starts with the PNG file signature (0x89504E470D0A1A0A).
func isPNG(data []byte) bool {
	return bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n"))
}

// decodePNGToGrayscale decodes a PNG byte slice into 8bpp grayscale raw pixels.
func decodePNGToGrayscale(data []byte) ([]byte, uint, uint, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to decode PNG: %w", err)
	}
	bounds := img.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()
	if imgW == 0 || imgH == 0 {
		return nil, 0, 0, fmt.Errorf("invalid PNG dimensions %dx%d", imgW, imgH)
	}

	raw := make([]byte, imgW*imgH)
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// Standard ITU-R BT.601 luma conversion
			raw[y*imgW+x] = byte((19595*r + 38469*g + 7472*b) >> 24)
		}
	}
	return raw, uint(imgW), uint(imgH), nil
}

// HandleExtractFMD processes raw image data and returns an extracted FMD.
func (h *FMDHandler) HandleExtractFMD(w http.ResponseWriter, r *http.Request) {
	var req dto.ExtractFMDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Invalid JSON payload: "+err.Error())
		return
	}

	if req.RawImageBase64 == "" {
		dto.Error(w, http.StatusBadRequest, 400, "raw_image_base64 field is required.")
		return
	}
	if req.DPI == 0 {
		req.DPI = 500 // Standard optical reader resolution
	}

	rawBytes, err := base64.StdEncoding.DecodeString(req.RawImageBase64)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Failed to decode base64 raw image: "+err.Error())
		return
	}

	width := req.Width
	height := req.Height
	if isPNG(rawBytes) {
		gray, imgW, imgH, err := decodePNGToGrayscale(rawBytes)
		if err != nil {
			dto.Error(w, http.StatusBadRequest, 400, "Failed to decode PNG image: "+err.Error())
			return
		}
		rawBytes = gray
		width = imgW
		height = imgH
	} else if width == 0 || height == 0 {
		dto.Error(w, http.StatusBadRequest, 400, "width and height are required for raw uncompressed image data.")
		return
	}

	fmdFmt, err := dp.ParseFMDFormat(req.Format)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, err.Error())
		return
	}

	fmd, err := dp.CreateFMDFromRaw(rawBytes, width, height, req.DPI, dp.FingerPosition(req.FingerPosition), fmdFmt)
	if err != nil {
		dto.Error(w, http.StatusUnprocessableEntity, 422, "Minutiae extraction failed: "+err.Error())
		return
	}

	dto.JSON(w, http.StatusOK, map[string]interface{}{
		"fmd_base64": base64.StdEncoding.EncodeToString(fmd),
		"fmd_size":   len(fmd),
		"format":     fmdFmt.String(),
	})
}

// rawToJPEGBase64 encodes an 8bpp grayscale raw buffer (len == width*height) as a Base64 JPEG preview.
func rawToJPEGBase64(raw []byte, width, height uint) (string, error) {
	img := &image.Gray{
		Pix:    raw,
		Stride: int(width),
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// HandleEnroll fuses 2+ RAW captures into one DP REGISTRATION FMD ("AP..." in base64).
// Each sample is a DigitalPersona WebSDK capture (JSON object or JSON string), or a plain
// base64/base64url string of raw 8bpp pixels with top-level width/height. Re-encoded images
// (PNG/JPEG) change the pixel data and are rejected.
func (h *FMDHandler) HandleEnroll(w http.ResponseWriter, r *http.Request) {
	var req dto.EnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Invalid JSON payload: "+err.Error())
		return
	}

	if len(req.Samples) < 2 {
		dto.Error(w, http.StatusBadRequest, 400, "Enrollment requires at least 2 raw captures of the same finger.")
		return
	}

	samples := make([]dp.RawSample, 0, len(req.Samples))
	for i, item := range req.Samples {
		s, err := parseRawSample(item, req.Width, req.Height, req.DPI)
		if err != nil {
			dto.Error(w, http.StatusBadRequest, 400, fmt.Sprintf("Sample %d: %v", i, err))
			return
		}
		samples = append(samples, s)
	}

	enrolledFMD, err := dp.CreateEnrollmentFromRaw(samples, dp.FingerPosition(req.FingerPosition))
	if err != nil {
		dto.Error(w, http.StatusUnprocessableEntity, 422, "Enrollment failed: "+err.Error())
		return
	}

	// Preview image: JPEG of the last raw capture.
	last := samples[len(samples)-1]
	imageB64, err := rawToJPEGBase64(last.Data, last.Width, last.Height)
	if err != nil {
		dto.Error(w, http.StatusInternalServerError, 500, "Failed to encode preview image: "+err.Error())
		return
	}

	dto.JSON(w, http.StatusOK, map[string]interface{}{
		"session_id":          req.SessionID,
		"enrolled_fmd_base64": base64.StdEncoding.EncodeToString(enrolledFMD),
		"enrolled_fmd_size":   len(enrolledFMD),
		"samples_processed":   len(samples),
		"format":              dp.FMDFormatDPRegFeatures.String(),
		"image_base64":        imageB64,
		"image_format":        "jpeg",
		"image_width":         last.Width,
		"image_height":        last.Height,
	})
}

// HandleCompare performs 1:1 fingerprint verification between probe and enrolled FMD.
func (h *FMDHandler) HandleCompare(w http.ResponseWriter, r *http.Request) {
	var req dto.CompareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Invalid JSON payload: "+err.Error())
		return
	}

	if req.ProbeFMDBase64 == "" || req.EnrolledFMDBase64 == "" {
		dto.Error(w, http.StatusBadRequest, 400, "Both probe_fmd_base64 and enrolled_fmd_base64 are required.")
		return
	}

	probe, err := base64.StdEncoding.DecodeString(req.ProbeFMDBase64)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Failed to decode probe_fmd_base64: "+err.Error())
		return
	}

	enrolled, err := base64.StdEncoding.DecodeString(req.EnrolledFMDBase64)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Failed to decode enrolled_fmd_base64: "+err.Error())
		return
	}

	fmdFmt, err := dp.ParseFMDFormat(req.Format)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, err.Error())
		return
	}

	isMatch, score, err := dp.Compare(probe, enrolled, fmdFmt, req.Threshold)
	if err != nil {
		dto.Error(w, http.StatusUnprocessableEntity, 422, "Comparison failed: "+err.Error())
		return
	}

	appliedThreshold := req.Threshold
	if appliedThreshold == 0 {
		appliedThreshold = dp.DefaultDissimilarityThreshold
	}

	dto.JSON(w, http.StatusOK, map[string]interface{}{
		"match":               isMatch,
		"dissimilarity_score": score,
		"threshold_applied":   appliedThreshold,
		"format":              fmdFmt.String(),
	})
}

// HandleIdentify searches a probe FMD across an array of candidate FMDs (1:N).
func (h *FMDHandler) HandleIdentify(w http.ResponseWriter, r *http.Request) {
	var req dto.IdentifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Invalid JSON payload: "+err.Error())
		return
	}

	if req.ProbeFMDBase64 == "" {
		dto.Error(w, http.StatusBadRequest, 400, "probe_fmd_base64 is required.")
		return
	}

	probe, err := base64.StdEncoding.DecodeString(req.ProbeFMDBase64)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Failed to decode probe_fmd_base64: "+err.Error())
		return
	}

	candidates := make([][]byte, 0, len(req.CandidateFMDsBase64))
	for _, c := range req.CandidateFMDsBase64 {
		b, err := base64.StdEncoding.DecodeString(c)
		if err != nil {
			continue
		}
		candidates = append(candidates, b)
	}

	fmdFmt, err := dp.ParseFMDFormat(req.Format)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, err.Error())
		return
	}

	matches, err := dp.Identify(probe, candidates, fmdFmt, req.Threshold)
	if err != nil {
		dto.Error(w, http.StatusUnprocessableEntity, 422, "Identification failed: "+err.Error())
		return
	}

	dto.JSON(w, http.StatusOK, map[string]interface{}{
		"match_found":         len(matches) > 0,
		"matches":             matches,
		"candidates_searched": len(candidates),
	})
}

// HandleConvert converts an FMD from one format to another (e.g. ANSI 378 to ISO 19794-2).
func (h *FMDHandler) HandleConvert(w http.ResponseWriter, r *http.Request) {
	var req dto.ConvertFMDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Invalid JSON payload: "+err.Error())
		return
	}

	if req.FMDBase64 == "" {
		dto.Error(w, http.StatusBadRequest, 400, "fmd_base64 is required.")
		return
	}

	raw, err := base64.StdEncoding.DecodeString(req.FMDBase64)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Failed to decode fmd_base64: "+err.Error())
		return
	}

	fromFmt, err := dp.ParseFMDFormat(req.FromFormat)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "from_format error: "+err.Error())
		return
	}

	toFmt, err := dp.ParseFMDFormat(req.ToFormat)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "to_format error: "+err.Error())
		return
	}

	converted, err := dp.ConvertFMD(raw, fromFmt, toFmt)
	if err != nil {
		dto.Error(w, http.StatusUnprocessableEntity, 422, "Conversion failed: "+err.Error())
		return
	}

	dto.JSON(w, http.StatusOK, map[string]interface{}{
		"fmd_base64":  base64.StdEncoding.EncodeToString(converted),
		"from_format": fromFmt.String(),
		"to_format":   toFmt.String(),
		"fmd_size":    len(converted),
	})
}

// HandleQuality evaluates NFIQ quality score for a raw fingerprint image.
func (h *FMDHandler) HandleQuality(w http.ResponseWriter, r *http.Request) {
	var req dto.NFIQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Invalid JSON payload: "+err.Error())
		return
	}

	if req.RawImageBase64 == "" {
		dto.Error(w, http.StatusBadRequest, 400, "raw_image_base64 is required.")
		return
	}
	if req.DPI == 0 {
		req.DPI = 500
	}

	raw, err := base64.StdEncoding.DecodeString(req.RawImageBase64)
	if err != nil {
		dto.Error(w, http.StatusBadRequest, 400, "Failed to decode raw_image_base64: "+err.Error())
		return
	}

	width := req.Width
	height := req.Height
	if isPNG(raw) {
		gray, imgW, imgH, err := decodePNGToGrayscale(raw)
		if err != nil {
			dto.Error(w, http.StatusBadRequest, 400, "Failed to decode PNG image: "+err.Error())
			return
		}
		raw = gray
		width = imgW
		height = imgH
	} else if width == 0 || height == 0 {
		dto.Error(w, http.StatusBadRequest, 400, "width and height are required.")
		return
	}

	score, err := dp.GetNFIQQualityFromRaw(raw, width, height, req.DPI)
	if err != nil {
		dto.Error(w, http.StatusUnprocessableEntity, 422, "Quality evaluation failed: "+err.Error())
		return
	}

	interpretations := map[uint]string{
		1: "Excellent (NFIQ 1)",
		2: "Very Good (NFIQ 2)",
		3: "Good (NFIQ 3)",
		4: "Fair (NFIQ 4)",
		5: "Poor (NFIQ 5)",
	}

	dto.JSON(w, http.StatusOK, map[string]interface{}{
		"nfiq_score":     score,
		"interpretation": interpretations[score],
	})
}
