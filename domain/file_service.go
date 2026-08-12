package domain

type FileService interface {
	Move(src, dst string) error
	Exists(path string) bool
	Size(path string) (int64, error)
	Remove(path string) error
	RemoveAll(path string) error
	ListFiles(dir string) ([]string, error)
}
