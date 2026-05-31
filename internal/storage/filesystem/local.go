package filesystem

import (
	"io"
	"os"
	"path/filepath"
)

type LocalDriver struct {
	root string
}

func NewLocalDriver(root string) *LocalDriver {
	return &LocalDriver{
		root: root,
	}
}
func (d *LocalDriver) Save(
	path string,
	r io.Reader,
) error {

	fullPath := filepath.Join(
		d.root,
		path,
	)

	err := os.MkdirAll(
		filepath.Dir(fullPath),
		0755,
	)

	if err != nil {
		return err
	}

	file, err := os.Create(fullPath)

	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(file, r)

	return err
}

func (d *LocalDriver) Open(
	path string,
) (io.ReadCloser, error) {

	return os.Open(
		filepath.Join(d.root, path),
	)
}

func (d *LocalDriver) Delete(
	path string,
) error {

	return os.Remove(
		filepath.Join(d.root, path),
	)
}