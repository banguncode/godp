package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"godp/pkg/dp"
)

// webSDKSample is a DigitalPersona WebSDK "Raw" capture:
//
//	{"Compression":0,"Data":"<base64url>","Format":{"iWidth":..,"iHeight":..,"iXdpi":..,"uBPP":8,...},"Header":{...},"Version":1}
type webSDKSample struct {
	Compression json.RawMessage `json:"Compression"`
	Data        string          `json:"Data"`
	Format      struct {
		Width  uint `json:"iWidth"`
		Height uint `json:"iHeight"`
		XDPI   uint `json:"iXdpi"`
		BPP    uint `json:"uBPP"`
	} `json:"Format"`
}

// decodeBase64Flexible decodes standard base64 or base64url, with or without '=' padding.
// WebSDK sends base64url ('-' and '_'), which base64.StdEncoding rejects.
func decodeBase64Flexible(s string) ([]byte, error) {
	s = strings.NewReplacer("\r", "", "\n", "", " ", "").Replace(s)
	s = strings.NewReplacer("-", "+", "_", "/").Replace(s)
	s = strings.TrimRight(s, "=")
	return base64.RawStdEncoding.DecodeString(s)
}

// trimToFrame returns exactly width*height pixels, dropping the trailing padding
// that real captures carry (same as PHP-DP's RawSample::fromWebSdkJson).
func trimToFrame(raw []byte, width, height uint) ([]byte, error) {
	if width == 0 || height == 0 {
		return nil, errors.New("width and height must be greater than zero")
	}
	need := int(width * height)
	if len(raw) < need {
		return nil, fmt.Errorf("data (%d bytes) is shorter than width*height (%d)", len(raw), need)
	}
	return raw[:need], nil
}

func compressionIsNone(c json.RawMessage) bool {
	v := strings.ToLower(strings.Trim(strings.TrimSpace(string(c)), `"`))
	return v == "" || v == "null" || v == "0" || v == "none"
}

func parseWebSDKSample(data []byte) (dp.RawSample, error) {
	var ws webSDKSample
	if err := json.Unmarshal(data, &ws); err != nil {
		return dp.RawSample{}, fmt.Errorf("invalid WebSDK sample JSON: %w", err)
	}
	if ws.Data == "" {
		return dp.RawSample{}, errors.New("WebSDK sample has no Data field")
	}
	if !compressionIsNone(ws.Compression) {
		return dp.RawSample{}, fmt.Errorf("compressed samples are not supported (Compression=%s)", string(ws.Compression))
	}
	if ws.Format.BPP != 0 && ws.Format.BPP != 8 {
		return dp.RawSample{}, fmt.Errorf("only 8bpp samples are supported (uBPP=%d)", ws.Format.BPP)
	}

	raw, err := decodeBase64Flexible(ws.Data)
	if err != nil {
		return dp.RawSample{}, fmt.Errorf("Data is not valid base64/base64url: %w", err)
	}
	raw, err = trimToFrame(raw, ws.Format.Width, ws.Format.Height)
	if err != nil {
		return dp.RawSample{}, err
	}

	dpi := ws.Format.XDPI
	if dpi == 0 {
		dpi = 500
	}
	return dp.RawSample{Data: raw, Width: ws.Format.Width, Height: ws.Format.Height, DPI: dpi}, nil
}

// parseRawSample accepts one enrollment sample in any of these forms:
//   - a WebSDK capture JSON object (width/height/dpi are read from its Format)
//   - a JSON string containing that object
//   - a plain base64/base64url string of raw 8bpp pixels (needs defW/defH from the request)
func parseRawSample(item json.RawMessage, defW, defH, defDPI uint) (dp.RawSample, error) {
	trimmed := bytes.TrimSpace(item)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return dp.RawSample{}, errors.New("sample is empty")
	}

	var plain string
	if trimmed[0] == '"' {
		if err := json.Unmarshal(trimmed, &plain); err != nil {
			return dp.RawSample{}, fmt.Errorf("invalid sample string: %w", err)
		}
		plain = strings.TrimSpace(plain)
		if strings.HasPrefix(plain, "{") {
			trimmed = []byte(plain)
		}
	}

	if trimmed[0] == '{' {
		return parseWebSDKSample(trimmed)
	}

	if plain == "" {
		return dp.RawSample{}, errors.New("unsupported sample type (expected a WebSDK capture object or a base64 string)")
	}
	if defW == 0 || defH == 0 {
		return dp.RawSample{}, errors.New("width and height are required for plain base64 samples")
	}
	raw, err := decodeBase64Flexible(plain)
	if err != nil {
		return dp.RawSample{}, fmt.Errorf("invalid base64: %w", err)
	}
	if isPNG(raw) {
		return dp.RawSample{}, errors.New("PNG samples are not accepted; send the raw capture (re-encoded images do not match reliably)")
	}
	raw, err = trimToFrame(raw, defW, defH)
	if err != nil {
		return dp.RawSample{}, err
	}
	if defDPI == 0 {
		defDPI = 500
	}
	return dp.RawSample{Data: raw, Width: defW, Height: defH, DPI: defDPI}, nil
}
