package common

import "github.com/pkg/errors"

var (
	ERR_LOCK_ALREADY_REQUIRED = errors.New("the lock is already occupied")
)
