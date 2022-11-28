package storage

import "errors"

var (
	ErrNoSuchKey        = errors.New("no such key")
	ErrNoSuchBucket     = errors.New("no such bucket")
	ErrKeyAlreadyExists = errors.New("key already exist")

	ErrUnsupportedMethod = errors.New("unsupported method")
)
