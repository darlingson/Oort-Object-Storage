package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func oortDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".oort")
}

func TokenPath() string {
	return filepath.Join(oortDir(), "token")
}

func ReadToken() (string, error) {
	data, err := os.ReadFile(TokenPath())
	if err != nil {
		return "", fmt.Errorf("not logged in: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func WriteToken(token string) error {
	dir := oortDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(TokenPath(), []byte(token+"\n"), 0600)
}

func DeleteToken() error {
	return os.Remove(TokenPath())
}
