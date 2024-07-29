package node

import "github.com/quic-go/quic-go"

type (
	ApplicationErrorCode = quic.ApplicationErrorCode
)

const (
	ErrCodeNoError              ApplicationErrorCode = 0x100
	ErrCodeGeneralProtocolError ApplicationErrorCode = 0x101
	ErrCodeInternalError        ApplicationErrorCode = 0x102
	ErrCodeStreamCreationError  ApplicationErrorCode = 0x103
	ErrCodeClosedCriticalStream ApplicationErrorCode = 0x104
	ErrCodeFrameUnexpected      ApplicationErrorCode = 0x105
	ErrCodeFrameError           ApplicationErrorCode = 0x106
	ErrCodeExcessiveLoad        ApplicationErrorCode = 0x107
	ErrCodeIDError              ApplicationErrorCode = 0x108
	ErrCodeSettingsError        ApplicationErrorCode = 0x109
	ErrCodeMissingSettings      ApplicationErrorCode = 0x10a
	ErrCodeRequestRejected      ApplicationErrorCode = 0x10b
	ErrCodeRequestCanceled      ApplicationErrorCode = 0x10c
	ErrCodeRequestIncomplete    ApplicationErrorCode = 0x10d
	ErrCodeMessageError         ApplicationErrorCode = 0x10e
	ErrCodeConnectError         ApplicationErrorCode = 0x10f
	ErrCodeVersionFallback      ApplicationErrorCode = 0x110
	ErrCodeDatagramError        ApplicationErrorCode = 0x33
)
