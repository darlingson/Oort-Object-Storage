package filesystem

import (
	"io"
)

type StorageDriver interface {
	Save(path string, r io.Reader) error
	Open(path string) (io.ReadCloser, error)
	Delete(path string) error
}