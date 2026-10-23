package dp

/*
#cgo CFLAGS: -I${SRCDIR}/../../lib/include
#cgo LDFLAGS: -L${SRCDIR}/../../lib/x64 -Wl,-rpath,$ORIGIN/../../lib/x64:$ORIGIN/../lib/x64:/usr/lib/Crossmatch/urusdk-linux/Linux/lib/x64 -ldpfj -ldpfpdd -lm
#include <dpfj.h>
#include <dpfj_quality.h>
#include <dpfj_compression.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
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

// CreateEnrollmentFMD combines multiple candidate FMD templates (typically 3 to 4 samples)
// into a single high-quality master enrolled template suitable for persistent storage.
func CreateEnrollmentFMD(samples [][]byte, fmdFmt FMDFormat) ([]byte, error) {
	if len(samples) < 2 {
		return nil, fmt.Errorf("dpfj: enrollment requires at least 2 samples, provided %d", len(samples))
	}

	enrollmentMutex.Lock()
	defer enrollmentMutex.Unlock()

	// 1. Start enrollment session
	rc := C.dpfj_start_enrollment(C.int(fmdFmt))
	if err := MapErrorCode(int(rc)); err != nil {
		return nil, fmt.Errorf("start enrollment failed: %w", err)
	}
	defer C.dpfj_finish_enrollment()

	// 2. Feed candidate FMD samples into the enrollment engine
	ready := false
	for i, sample := range samples {
		if len(sample) == 0 {
			return nil, fmt.Errorf("sample at index %d is empty", i)
		}

		addRc := C.dpfj_add_to_enrollment(
			C.int(fmdFmt),
			(*C.uchar)(unsafe.Pointer(&sample[0])),
			C.uint(len(sample)),
			0, // View index 0
		)

		const DPFJ_SUCCESS = 0
		const DPFJ_E_MORE_DATA = 0x0d | (0x05BA << 16)

		if addRc == DPFJ_SUCCESS {
			ready = true
		} else if addRc == DPFJ_E_MORE_DATA {
			// Engine needs more samples
			continue
		} else {
			return nil, fmt.Errorf("failed adding sample %d: %w", i+1, MapErrorCode(int(addRc)))
		}
	}

	if !ready && len(samples) < 4 {
		return nil, ErrEnrollmentNotReady
	}

	// 3. Create the aggregated master enrollment template
	fmdBuf := make([]byte, MaxFMDSize)
	fmdSize := C.uint(MaxFMDSize)

	createRc := C.dpfj_create_enrollment_fmd(
		(*C.uchar)(unsafe.Pointer(&fmdBuf[0])),
		&fmdSize,
	)

	if err := MapErrorCode(int(createRc)); err != nil {
		return nil, fmt.Errorf("create enrollment FMD failed: %w", err)
	}

	out := make([]byte, fmdSize)
	copy(out, fmdBuf[:fmdSize])
	return out, nil
}

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
