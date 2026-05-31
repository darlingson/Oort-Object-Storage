package filesystem

import "io"
import "os"

type StorageDriver interface {
	Save(file io.Reader) (string, error)
	Open(path string) (*os.File, error)
	Delete(path string) error
}