package security

import "errors"

var (
	ErrLocalSecretMissing = errors.New("local node has no secret configured")
	ErrPeerSecretMissing  = errors.New("peer has no secret configured")
	ErrSecretMismatch     = errors.New("different secrets configured")
)
