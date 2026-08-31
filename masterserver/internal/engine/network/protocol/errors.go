package protocol

import "errors"

var (
	ErrUnderflow       = errors.New("protocol reader underflow")
	ErrInvalidBool     = errors.New("invalid boolean")
	ErrValueTooLarge   = errors.New("value exceeds uint16 length prefix")
	ErrPayloadTooLarge = errors.New("payload exceeds configured maximum")
)
