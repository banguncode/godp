package dp

/*
#cgo CFLAGS: -I${SRCDIR}/../../lib/include
#cgo LDFLAGS: -L${SRCDIR}/../../lib/x64 -Wl,-rpath,$ORIGIN/../../lib/x64:$ORIGIN/../lib/x64:/usr/lib/Crossmatch/urusdk-linux/Linux/lib/x64 -ldpfj -ldpfpdd -lm
#include <dpfj.h>
#include <dpfpdd.h>
#include <dpfj_quality.h>
#include <dpfj_compression.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"
)

var (
	// enrollmentMutex ensures thread safety for dpfj enrollment sessions,
	// which maintain global session state inside the underlying C SDK.
	enrollmentMutex sync.Mutex
)

// CreateFMDFromRaw extracts minutiae from raw uncompressed 8bpp grayscale pixel data.
func CreateFMDFromRaw(imgData []byte, width, height, dpi uint, pos FingerPosition, fmt FMDFormat) ([]byte, error) {
	if len(imgData) == 0 {
		return nil, errors.New("dpfj: image data buffer is empty")
	}

	fmdBuf := make([]byte, MaxFMDSize)
	fmdSize := C.uint(MaxFMDSize)

	rc := C.dpfj_create_fmd_from_raw(
		(*C.uchar)(unsafe.Pointer(&imgData[0])),
		C.uint(len(imgData)),
		C.uint(width),
		C.uint(height),
		C.uint(dpi),
		C.int(pos),
		0, // CBEFF ID (0 = standard)
		C.int(fmt),
		(*C.uchar)(unsafe.Pointer(&fmdBuf[0])),
		&fmdSize,
	)

	if err := MapErrorCode(int(rc)); err != nil {
		return nil, err
	}

	out := make([]byte, fmdSize)
	copy(out, fmdBuf[:fmdSize])
	return out, nil
}

// CreateFMDFromFID extracts minutiae features from an ANSI 381 or ISO 19794-4 FID image buffer.
func CreateFMDFromFID(fidData []byte, fidFmt FIDFormat, fmdFmt FMDFormat) ([]byte, error) {
	if len(fidData) == 0 {
		return nil, errors.New("dpfj: FID buffer is empty")
	}

	fmdBuf := make([]byte, MaxFMDSize)
	fmdSize := C.uint(MaxFMDSize)

	rc := C.dpfj_create_fmd_from_fid(
		C.int(fidFmt),
		(*C.uchar)(unsafe.Pointer(&fidData[0])),
		C.uint(len(fidData)),
		C.int(fmdFmt),
		(*C.uchar)(unsafe.Pointer(&fmdBuf[0])),
		&fmdSize,
	)

	if err := MapErrorCode(int(rc)); err != nil {
		return nil, err
	}

	out := make([]byte, fmdSize)
	copy(out, fmdBuf[:fmdSize])
	return out, nil
}

// RawSample is one uncompressed 8bpp grayscale capture of the same finger.
type RawSample struct {
	Data   []byte
	Width  uint
	Height uint
	DPI    uint
}

var (
	initOnce sync.Once
	initErr  error
)

// Init loads the reader-driver layer and selects the matching engine.
// The DP enrollment functions return NOT_IMPLEMENTED unless dpfpdd_init()
// ran first and the DPFJ engine is selected (same order as DPUruNet.dll).
// Safe to call many times; it is invoked automatically by CreateEnrollmentFromRaw.
func Init() error {
	initOnce.Do(func() {
		// rc 0 (SUCCESS) or 1 (already initialized) are both OK; retry transient
		// failures a few times, like PHP-DP's NativeMethods does.
		var rc C.int
		for attempt := 0; attempt < 5; attempt++ {
			rc = C.dpfpdd_init()
			if rc == 0 || rc == 1 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if rc != 0 && rc != 1 {
			initErr = fmt.Errorf("dpfpdd_init failed: %w", MapErrorCode(int(rc)))
			return
		}
		if rc := C.dpfj_select_engine(nil, C.DPFJ_ENGINE_DPFJ); rc != 0 {
			initErr = fmt.Errorf("dpfj_select_engine failed: %w", MapErrorCode(int(rc)))
		}
	})
	return initErr
}

// CreateEnrollmentFromRaw fuses 2 or more raw captures of one finger into a
// single DigitalPersona REGISTRATION template (base64 starts with "AP...").
//
// Rules taken from the vendor SDK behaviour (see PHP-DP docs/troubleshooting.md):
//   - dpfj_start_enrollment() takes DP_REG_FEATURES (1), NOT ISO/ANSI.
//   - every sample is a DP_PRE_REG_FEATURES (0) FMD.
//   - each FMD is extracted INSIDE the started session, interleaved with
//     dpfj_add_to_enrollment(), otherwise samples are accepted but never fuse.
//   - MORE_DATA from add_to_enrollment is normal, not an error.
func CreateEnrollmentFromRaw(samples []RawSample, pos FingerPosition) ([]byte, error) {
	if len(samples) < 2 {
		return nil, fmt.Errorf("dpfj: enrollment requires at least 2 samples, provided %d", len(samples))
	}
	if err := Init(); err != nil {
		return nil, err
	}

	enrollmentMutex.Lock()
	defer enrollmentMutex.Unlock()

	rc := C.dpfj_start_enrollment(C.int(FMDFormatDPRegFeatures))
	if err := MapErrorCode(int(rc)); err != nil {
		return nil, fmt.Errorf("start enrollment failed: %w", err)
	}
	defer C.dpfj_finish_enrollment()

	for i, s := range samples {
		if len(s.Data) == 0 {
			return nil, fmt.Errorf("sample at index %d is empty", i)
		}

		pre, err := CreateFMDFromRaw(s.Data, s.Width, s.Height, s.DPI, pos, FMDFormatDPPreRegFeatures)
		if err != nil {
			return nil, fmt.Errorf("extract sample %d failed: %w", i+1, err)
		}

		addRc := C.dpfj_add_to_enrollment(
			C.int(FMDFormatDPPreRegFeatures),
			(*C.uchar)(unsafe.Pointer(&pre[0])),
			C.uint(len(pre)),
			0,
		)
		if addRc == 0 {
			break // SUCCESS: fusion is complete, remaining samples are not needed (same as PHP-DP)
		}
		if int(addRc) != dpfjEMoreData {
			return nil, fmt.Errorf("failed adding sample %d: %w", i+1, MapErrorCode(int(addRc)))
		}
		// MORE_DATA: normal "keep feeding samples" signal, not an error.
	}

	// Always try to finalize; the SDK decides if the set is sufficient.
	fmdBuf := make([]byte, MaxFMDSize)
	fmdSize := C.uint(MaxFMDSize)
	createRc := C.dpfj_create_enrollment_fmd((*C.uchar)(unsafe.Pointer(&fmdBuf[0])), &fmdSize)
	if err := MapErrorCode(int(createRc)); err != nil {
		return nil, fmt.Errorf("create enrollment FMD failed: %w", err)
	}

	out := make([]byte, fmdSize)
	copy(out, fmdBuf[:fmdSize])
	return out, nil
}

const dpfjEMoreData = 0x0d | (0x05BA << 16)

// Compare performs 1:1 verification between a probe FMD and an enrolled FMD.
// Returns isMatch (true if score <= threshold) and the exact dissimilarity score (0 = perfect match).
func Compare(probeFMD, enrolledFMD []byte, fmdFmt FMDFormat, threshold uint) (bool, uint, error) {
	if len(probeFMD) == 0 || len(enrolledFMD) == 0 {
		return false, 0, errors.New("dpfj: probe or enrolled FMD buffer is empty")
	}

	if threshold == 0 {
		threshold = DefaultDissimilarityThreshold
	}

	var score C.uint

	rc := C.dpfj_compare(
		C.int(fmdFmt),
		(*C.uchar)(unsafe.Pointer(&probeFMD[0])),
		C.uint(len(probeFMD)),
		0, // probe view index
		C.int(fmdFmt),
		(*C.uchar)(unsafe.Pointer(&enrolledFMD[0])),
		C.uint(len(enrolledFMD)),
		0, // enrolled view index
		&score,
	)

	if err := MapErrorCode(int(rc)); err != nil {
		return false, 0, err
	}

	dissimilarityScore := uint(score)
	isMatch := dissimilarityScore <= threshold
	return isMatch, dissimilarityScore, nil
}

// Identify performs 1:N biometric identification by searching a probe FMD across candidate FMDs.
// Returns a slice of Candidates meeting the dissimilarity threshold, ranked by confidence.
func Identify(probeFMD []byte, candidates [][]byte, fmdFmt FMDFormat, threshold uint) ([]Candidate, error) {
	if len(probeFMD) == 0 {
		return nil, errors.New("dpfj: probe FMD buffer is empty")
	}
	if len(candidates) == 0 {
		return []Candidate{}, nil
	}

	if threshold == 0 {
		threshold = DefaultDissimilarityThreshold
	}

	var matched []Candidate
	for idx, cand := range candidates {
		if len(cand) == 0 {
			continue
		}
		isMatch, score, err := Compare(probeFMD, cand, fmdFmt, threshold)
		if err != nil {
			continue
		}
		if isMatch {
			matched = append(matched, Candidate{
				Index: uint(idx),
				Score: score,
			})
		}
	}

	return matched, nil
}

// ConvertFMD transforms an FMD from one standard format to another (e.g. ANSI 378 to ISO 19794-2).
func ConvertFMD(inFMD []byte, inFmt FMDFormat, outFmt FMDFormat) ([]byte, error) {
	if len(inFMD) == 0 {
		return nil, errors.New("dpfj: input FMD buffer is empty")
	}

	outBuf := make([]byte, MaxFMDSize)
	outSize := C.uint(MaxFMDSize)

	rc := C.dpfj_fmd_convert(
		C.int(inFmt),
		(*C.uchar)(unsafe.Pointer(&inFMD[0])),
		C.uint(len(inFMD)),
		C.int(outFmt),
		(*C.uchar)(unsafe.Pointer(&outBuf[0])),
		&outSize,
	)

	if err := MapErrorCode(int(rc)); err != nil {
		return nil, err
	}

	out := make([]byte, outSize)
	copy(out, outBuf[:outSize])
	return out, nil
}

// GetNFIQQualityFromFID assesses fingerprint quality according to NIST NFIQ standards (1 = best, 5 = worst).
func GetNFIQQualityFromFID(fidData []byte, fidFmt FIDFormat) (uint, error) {
	if len(fidData) == 0 {
		return 0, errors.New("dpfj: FID buffer is empty")
	}

	var score C.uint
	rc := C.dpfj_quality_nfiq_from_fid(
		C.int(fidFmt),
		(*C.uchar)(unsafe.Pointer(&fidData[0])),
		C.uint(len(fidData)),
		0, // View index 0
		C.int(1), // DPFJ_QUALITY_NFIQ_NIST
		&score,
	)

	if err := MapErrorCode(int(rc)); err != nil {
		return 0, err
	}

	return uint(score), nil
}

// GetNFIQQualityFromRaw assesses fingerprint quality from uncompressed raw 8bpp grayscale image data.
func GetNFIQQualityFromRaw(imgData []byte, width, height, dpi uint) (uint, error) {
	if len(imgData) == 0 {
		return 0, errors.New("dpfj: raw image buffer is empty")
	}

	var score C.uint
	rc := C.dpfj_quality_nfiq_from_raw(
		(*C.uchar)(unsafe.Pointer(&imgData[0])),
		C.uint(len(imgData)),
		C.uint(width),
		C.uint(height),
		C.uint(dpi),
		8, // 8 bits per pixel
		C.int(1), // DPFJ_QUALITY_NFIQ_NIST
		&score,
	)

	if err := MapErrorCode(int(rc)); err != nil {
		return 0, err
	}

	return uint(score), nil
}
