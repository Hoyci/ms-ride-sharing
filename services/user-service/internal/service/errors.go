package service

import "errors"

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	InternalServerError  = errors.New("internal server error")
)
