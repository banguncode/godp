package handler

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseRawSampleWebSDKObject(t *testing.T) {
	pixels := []byte{1, 2, 3, 4}
	padded := append(append([]byte{}, pixels...), 0, 0, 0)
	data := strings.NewReplacer("+", "-", "/", "_").Replace(base64.RawStdEncoding.EncodeToString(padded))

	obj, _ := json.Marshal(map[string]interface{}{
		"Compression": 0,
		"Data":        data,
		"Format":      map[string]interface{}{"iWidth": 2, "iHeight": 2, "iXdpi": 700, "uBPP": 8},
	})

	for name, in := range map[string]json.RawMessage{
		"object":      obj,
		"json string": mustMarshal(string(obj)),
	} {
		s, err := parseRawSample(in, 0, 0, 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(s.Data) != string(pixels) || s.Width != 2 || s.Height != 2 || s.DPI != 700 {
			t.Fatalf("%s: unexpected sample %+v", name, s)
		}
	}
}

func TestParseRawSampleRejectsShortData(t *testing.T) {
	obj := `{"Compression":0,"Data":"AQI","Format":{"iWidth":10,"iHeight":10,"iXdpi":500}}`
	if _, err := parseRawSample(json.RawMessage(obj), 0, 0, 0); err == nil {
		t.Fatal("expected error for data shorter than width*height")
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
