package dp

import (
	"errors"
	"fmt"
)

var (
	ErrNotImplemented        = errors.New("dpfj: API call or requested algorithm is not implemented")
	ErrFailure               = errors.New("dpfj: unspecified internal SDK failure")
	ErrNoData                = errors.New("dpfj: no data available")
	ErrMoreData              = errors.New("dpfj: memory buffer too small or more data required")
	ErrInvalidParameter      = errors.New("dpfj: one or more parameters are invalid")
	ErrInvalidFID            = errors.New("dpfj: image data (FID) is invalid or corrupted")
	ErrTooSmallArea          = errors.New("dpfj: image cropped area is too small to extract minutiae")
	ErrInvalidFMD            = errors.New("dpfj: template data (FMD) is invalid or corrupted")
	ErrEnrollmentInProgress  = errors.New("dpfj: another enrollment session is already in progress")
	ErrEnrollmentNotStarted  = errors.New("dpfj: enrollment session has not been started")
	ErrEnrollmentNotReady    = errors.New("dpfj: not enough samples added to complete enrollment")
	ErrEnrollmentInvalidSet  = errors.New("dpfj: samples provided for enrollment are inconsistent or do not match")
	ErrQualityTooFewMinutiae = errors.New("dpfj: too few minutiae detected to evaluate quality score")
	ErrQualityLibNotFound    = errors.New("dpfj: image quality assessment library not found")
)

// MapErrorCode converts a DigitalPersona C SDK return code to a standard Go error.
func MapErrorCode(rc int) error {
	const (
		DPFJ_SUCCESS                    = 0
		DP_FACILITY                     = 0x0001
		DPFJ_E_NOT_IMPLEMENTED          = 0x0a | (DP_FACILITY << 16)
		DPFJ_E_FAILURE                  = 0x0b | (DP_FACILITY << 16)
		DPFJ_E_NO_DATA                  = 0x0c | (DP_FACILITY << 16)
		DPFJ_E_MORE_DATA                = 0x0d | (DP_FACILITY << 16)
		DPFJ_E_INVALID_PARAMETER        = 0x14 | (DP_FACILITY << 16)
		DPFJ_E_INVALID_FID              = 0x65 | (DP_FACILITY << 16)
		DPFJ_E_TOO_SMALL_AREA           = 0x66 | (DP_FACILITY << 16)
		DPFJ_E_INVALID_FMD              = 0xc9 | (DP_FACILITY << 16)
		DPFJ_E_ENROLLMENT_IN_PROGRESS   = 0x12d | (DP_FACILITY << 16)
		DPFJ_E_ENROLLMENT_NOT_STARTED   = 0x12e | (DP_FACILITY << 16)
		DPFJ_E_ENROLLMENT_NOT_READY     = 0x12f | (DP_FACILITY << 16)
		DPFJ_E_ENROLLMENT_INVALID_SET   = 0x130 | (DP_FACILITY << 16)
		DPFJ_E_QUALITY_TOO_FEW_MINUTIA  = 0x15f | (DP_FACILITY << 16)
		DPFJ_E_QUALITY_LIB_NOT_FOUND    = 0x161 | (DP_FACILITY << 16)
	)

	switch rc {
	case DPFJ_SUCCESS:
		return nil
	case DPFJ_E_NOT_IMPLEMENTED:
		return ErrNotImplemented
	case DPFJ_E_FAILURE:
		return ErrFailure
	case DPFJ_E_NO_DATA:
		return ErrNoData
	case DPFJ_E_MORE_DATA:
		return ErrMoreData
	case DPFJ_E_INVALID_PARAMETER:
		return ErrInvalidParameter
	case DPFJ_E_INVALID_FID:
		return ErrInvalidFID
	case DPFJ_E_TOO_SMALL_AREA:
		return ErrTooSmallArea
	case DPFJ_E_INVALID_FMD:
		return ErrInvalidFMD
	case DPFJ_E_ENROLLMENT_IN_PROGRESS:
		return ErrEnrollmentInProgress
	case DPFJ_E_ENROLLMENT_NOT_STARTED:
		return ErrEnrollmentNotStarted
	case DPFJ_E_ENROLLMENT_NOT_READY:
		return ErrEnrollmentNotReady
	case DPFJ_E_ENROLLMENT_INVALID_SET:
		return ErrEnrollmentInvalidSet
	case DPFJ_E_QUALITY_TOO_FEW_MINUTIA:
		return ErrQualityTooFewMinutiae
	case DPFJ_E_QUALITY_LIB_NOT_FOUND:
		return ErrQualityLibNotFound
	default:
		return fmt.Errorf("dpfj: unexpected SDK error code (0x%08X)", rc)
	}
}
