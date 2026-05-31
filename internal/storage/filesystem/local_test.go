package filesystem

import (
	"io"
	"strings"
	"testing"
)

func TestSaveFile(t *testing.T) {

	dir := t.TempDir()

	driver := NewLocalDriver(dir)

	content := strings.NewReader(
		"hello world",
	)

	err := driver.Save(
		"test/file.txt",
		content,
	)

	if err != nil {
		t.Fatalf(
			"expected no error got %v",
			err,
		)
	}
}

func TestOpenFile(t *testing.T) {

	dir := t.TempDir()

	driver := NewLocalDriver(dir)

	_ = driver.Save(
		"test/file.txt",
		strings.NewReader("hello"),
	)

	file, err := driver.Open(
		"test/file.txt",
	)

	if err != nil {
		t.Fatal(err)
	}

	defer file.Close()

	data, _ := io.ReadAll(file)

	if string(data) != "hello" {
		t.Fatalf(
			"unexpected content",
		)
	}
}

func TestDeleteFile(t *testing.T) {

	dir := t.TempDir()

	driver := NewLocalDriver(dir)

	_ = driver.Save(
		"test/file.txt",
		strings.NewReader("hello"),
	)

	err := driver.Delete(
		"test/file.txt",
	)

	if err != nil {
		t.Fatal(err)
	}
}