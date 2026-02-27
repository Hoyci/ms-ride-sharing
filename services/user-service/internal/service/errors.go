package service

import "errors"

var (
	ErrUserAlreadyExists         = errors.New("user already exists")
	ErrInvalidCredentials        = errors.New("invalid user credentials")
	ErrInternalServer            = errors.New("internal server error")
	ErrInvalidToken              = errors.New("invalid token")
	ErrInvalidTokenType          = errors.New("invalid token type")
	ErrRefreshTokenReuseDetected = errors.New("refresh token reuse detected; session invalidated")
)
