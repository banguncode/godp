# godp — REST API Documentation

`godp` exposes a standard, high-performance RESTful API for biometric fingerprint operations. All requests and responses use JSON encoding.

---

## Table of Contents

1. [Response Envelope](#response-envelope)
2. [Endpoints](#endpoints)
   - [Health Probe (`GET /health`)](#1-health-probe)
   - [Extract FMD (`POST /api/v1/fmd/extract`)](#2-extract-fmd)
   - [Enrollment Session (`POST /api/v1/fmd/enroll`)](#3-enrollment-session)
   - [1:1 Verification (`POST /api/v1/fmd/compare`)](#4-11-verification)
   - [1:N Identification (`POST /api/v1/fmd/identify`)](#5-1n-identification)
   - [Format Conversion (`POST /api/v1/fmd/convert`)](#6-format-conversion)
   - [NFIQ Quality Assessment (`POST /api/v1/quality/nfiq`)](#7-nfiq-quality-assessment)
3. [Supported Formats](#supported-formats)
4. [Error Codes](#error-codes)

---

## Response Envelope

All API endpoints return a uniform JSON schema:

### Success Response (`200 OK`)
```json
{
  "success": true,
  "data": {
    "enrolled_fmd_base64": "...",
    "enrolled_fmd_size": 844
  }
}
```

### Error Response (`4xx` / `5xx`)
```json
{
  "success": false,
  "error": {
    "code": 422,
    "message": "Minutiae extraction failed: dpfj: too few minutiae detected"
  }
}
```

---

## Endpoints

### 1. Health Probe
Checks service liveness, uptime, and memory utilization.

* **Method:** `GET`
* **Path:** `/health`

```bash
curl -s http://localhost:5050/health | python3 -m json.tool
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "go_version": "go1.23.2",
    "goroutines": 5,
    "memory_alloc_mb": 0.24,
    "service": "godp-biometric-engine",
    "status": "UP",
    "uptime_sec": 120,
    "version": "1.0.0"
  }
}
```

---

### 2. Extract FMD
Extracts minutiae features from an uncompressed 8bpp grayscale raw image.

* **Method:** `POST`
* **Path:** `/api/v1/fmd/extract`

```bash
curl -s -X POST http://localhost:5050/api/v1/fmd/extract \
  -H "Content-Type: application/json" \
  -d '{
    "raw_image_base64": "<BASE64_IMAGE_STRING>",
    "width": 500,
    "height": 550,
    "dpi": 500,
    "finger_position": 2,
    "format": "ISO_19794_2_2005"
  }'
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "fmd_base64": "Rk1EAAAA...",
    "fmd_size": 844,
    "format": "ISO_19794_2_2005"
  }
}
```

---

### 3. Enrollment Session
Fuses **2 or more raw captures of the same finger** into a single DigitalPersona
*registration* template (`DP_REG_FEATURES`).
The response also includes a JPEG preview of the last capture.

* **Method:** `POST`
* **Path:** `/api/v1/fmd/enroll`

**Request fields**

| Field | Type | Required | Description |
|---|---|:---:|---|
| `samples` | array | yes | Minimum 2 raw captures of the same finger (3-4 recommended). Each item is a DigitalPersona WebSDK capture (see below), or a plain base64 string of raw 8bpp pixels. |
| `width`, `height` | uint | plain base64 only | Pixel dimensions for plain base64 samples. Ignored for WebSDK captures (read from `Format`). |
| `dpi` | uint | no | Resolution for plain base64 samples (default `500`). WebSDK captures use `Format.iXdpi`. |
| `finger_position` | int | no | NIST/ISO finger position (default `0` = unknown). |
| `session_id` | string | no | Echoed back in the response. |
| `format` | string | no | **Ignored.** Enrollment output is always `DP_REG_FEATURES`. |

**WebSDK capture** (send it exactly as the SDK delivers it, as a JSON object or as a JSON string):
```json
{
  "Compression": 0,
  "Data": "_v769_Xz9_r8-vXz9fT1_...",
  "Format": { "iWidth": 500, "iHeight": 550, "iXdpi": 700, "uBPP": 8 },
  "Header": { "...": "ignored" },
  "Version": 1
}
```
- `Data` is **base64url** (`-` and `_`), with or without `=` padding. Standard base64 is accepted too.
- Trailing padding bytes are trimmed to exactly `iWidth * iHeight` bytes.
- Width, height and DPI are taken from `Format` (`iWidth`, `iHeight`, `iXdpi`).
- Compressed captures (`Compression` other than `0`/`none`) and non-8bpp captures are rejected.

> **Important:** send the raw pixels exactly as delivered by the reader.
> PNG/JPEG samples and pre-extracted FMDs (ISO/ANSI) are rejected, because
> re-encoded images do not match reliably.

```bash
curl -s -X POST http://localhost:5050/api/v1/fmd/enroll \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "emp-10492-enroll",
    "samples": [ <WEBSDK_CAPTURE_1>, <WEBSDK_CAPTURE_2>, <WEBSDK_CAPTURE_3>, <WEBSDK_CAPTURE_4> ]
  }'
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "session_id": "emp-10492-enroll",
    "enrolled_fmd_base64": "AP...",
    "enrolled_fmd_size": 1288,
    "samples_processed": 4,
    "format": "DP_REG_FEATURES",
    "image_base64": "/9j/4AAQSkZJRg...",
    "image_format": "jpeg",
    "image_width": 500,
    "image_height": 550
  }
}
```

`image_base64` is a Base64 JPEG of the **last** raw sample. Use it as
`<img src="data:image/jpeg;base64,...">`.

**Errors**

| HTTP | Cause |
|:---:|---|
| 400 | Fewer than 2 samples, or a sample that cannot be parsed: invalid JSON/base64, `Data` shorter than `iWidth * iHeight`, compressed or non-8bpp capture, PNG, missing `width`/`height` for plain base64. The message names the sample index. |
| 422 | Minutiae extraction failure, or the SDK could not build a template from the samples (e.g. captures do not match). |
| 500 | Failed to encode the preview image. |

---

### 4. 1:1 Verification
Compares a newly scanned probe FMD against a stored enrolled FMD (e.g. at user login).

* **Method:** `POST`
* **Path:** `/api/v1/fmd/compare`

```bash
curl -s -X POST http://localhost:5050/api/v1/fmd/compare \
  -H "Content-Type: application/json" \
  -d '{
    "probe_fmd_base64": "<LOGIN_PROBE_FMD>",
    "enrolled_fmd_base64": "<DATABASE_STORED_FMD>",
    "format": "ISO_19794_2_2005",
    "threshold": 21474
  }'
```

**Response 200 — Match Found:**
```json
{
  "success": true,
  "data": {
    "match": true,
    "dissimilarity_score": 0,
    "threshold_applied": 21474,
    "format": "ISO_19794_2_2005"
  }
}
```

---

### 5. 1:N Identification
Searches a probe FMD across an array of enrolled FMDs to identify the user.

* **Method:** `POST`
* **Path:** `/api/v1/fmd/identify`

```bash
curl -s -X POST http://localhost:5050/api/v1/fmd/identify \
  -H "Content-Type: application/json" \
  -d '{
    "probe_fmd_base64": "<PROBE_FMD>",
    "candidate_fmds_base64": [
      "<CANDIDATE_0>",
      "<CANDIDATE_1>",
      "<CANDIDATE_2>"
    ],
    "format": "ISO_19794_2_2005",
    "threshold": 21474
  }'
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "match_found": true,
    "candidates_searched": 3,
    "matches": [
      { "index": 1, "score": 12 }
    ]
  }
}
```

---

### 6. Format Conversion
Converts an FMD between ANSI INCITS 378 and ISO/IEC 19794-2.

* **Method:** `POST`
* **Path:** `/api/v1/fmd/convert`

```bash
curl -s -X POST http://localhost:5050/api/v1/fmd/convert \
  -H "Content-Type: application/json" \
  -d '{
    "fmd_base64": "<INPUT_ANSI_FMD>",
    "from_format": "ANSI_378_2004",
    "to_format": "ISO_19794_2_2005"
  }'
```

---

### 7. NFIQ Quality Assessment
Calculates the NIST NFIQ quality score (1 = Excellent, 5 = Poor).

* **Method:** `POST`
* **Path:** `/api/v1/quality/nfiq`

```bash
curl -s -X POST http://localhost:5050/api/v1/quality/nfiq \
  -H "Content-Type: application/json" \
  -d '{
    "raw_image_base64": "<RAW_IMAGE_BYTES>",
    "width": 500,
    "height": 550,
    "dpi": 500
  }'
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "nfiq_score": 1,
    "interpretation": "Excellent (NFIQ 1)"
  }
}
```

---

## Supported Formats

| Format Identifier | Specification | Description |
|---|---|---|
| `ISO_19794_2_2005` | ISO/IEC 19794-2:2005 | International biometric interchange format *(Recommended)* |
| `ANSI_378_2004` | ANSI INCITS 378-2004 | US NIST biometric interchange format |
| `DP_REG_FEATURES` | DigitalPersona Proprietary | Registration template produced by `/fmd/enroll` |
