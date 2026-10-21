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
Combines multiple candidate samples (3 to 4 recommended) into a single master enrollment template.

* **Method:** `POST`
* **Path:** `/api/v1/fmd/enroll`

```bash
curl -s -X POST http://localhost:5050/api/v1/fmd/enroll \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "emp-10492-enroll",
    "format": "ISO_19794_2_2005",
    "samples": [
      "<SAMPLE_1_FMD_BASE64>",
      "<SAMPLE_2_FMD_BASE64>",
      "<SAMPLE_3_FMD_BASE64>",
      "<SAMPLE_4_FMD_BASE64>"
    ]
  }'
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "session_id": "emp-10492-enroll",
    "enrolled_fmd_base64": "Rk1EAAAA...",
    "enrolled_fmd_size": 1288,
    "samples_processed": 4,
    "format": "ISO_19794_2_2005"
  }
}
```

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
| `DP_REG_FEATURES` | DigitalPersona Proprietary | Legacy registration template |
