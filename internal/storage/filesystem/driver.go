type StorageDriver interface {
    Save(reader io.Reader) (string, error)
    Delete(path string) error
    Open(path string) (*os.File, error)
}