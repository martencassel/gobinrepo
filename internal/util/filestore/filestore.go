package filestore

import (
	"os"
	"path/filepath"
)

type FileStore interface {
	Get(repoKey, path string) (digest string, found bool, err error)
	Put(repoKey, path, digest string) error
	Delete(repoKey, path string) error
	Exists(repoKey, path string) (bool, error)
	List(repoKey string) ([]Mapping, error)
}

type Mapping struct {
	Path   string
	Digest string
}

type fileStoreImpl struct {
	BasePath string
}

func NewFileStore(basePath string) FileStore {
	err := os.MkdirAll(basePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return &fileStoreImpl{
		BasePath: basePath,
	}
}

func (fs *fileStoreImpl) Put(repoKey, path, digest string) error {
	// 1. Build the target file path
	targetPath := filepath.Join(fs.BasePath, repoKey, path)
	dir := filepath.Dir(targetPath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// 2. Create a temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, "tmpfile-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	// 3. Write the digest
	if _, err := tmpFile.WriteString(digest + "\n"); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return err
	}

	// 4. Flush and close
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	// 5. Atomically rename into place
	if err := os.Rename(tmpName, targetPath); err != nil {
		os.Remove(tmpName)
		return err
	}

	// 6. (Optional) fsync the directory to ensure rename durability
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}

	return nil
}

func (fs *fileStoreImpl) Get(repoKey, path string) (string, bool, error) {
	targetPath := filepath.Join(fs.BasePath, repoKey, path)
	data, err := os.ReadFile(targetPath)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}

func (fs *fileStoreImpl) Delete(repoKey, path string) error {
	targetPath := filepath.Join(fs.BasePath, repoKey, path)
	return os.Remove(targetPath)
}

func (fs *fileStoreImpl) Exists(repoKey, path string) (bool, error) {
	targetPath := filepath.Join(fs.BasePath, repoKey, path)
	_, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (fs *fileStoreImpl) List(repoKey string) ([]Mapping, error) {
	var mappings []Mapping
	baseDir := filepath.Join(fs.BasePath, repoKey)
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		path = relPath
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mappings = append(mappings, Mapping{
			Path:   path,
			Digest: string(data),
		})
		return nil
	})
	if os.IsNotExist(err) {
		return mappings, nil
	}
	if err != nil {
		return nil, err
	}
	return mappings, nil
}
