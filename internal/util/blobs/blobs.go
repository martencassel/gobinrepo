package blobs

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"

	digest "github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
)

// BlobStore defines opaque binary object storage.
// Keys are typically digests or content hashes.
type BlobStore interface {
	// Put stores a blob from r under the given digest.
	Put(ctx context.Context, d digest.Digest, r io.Reader) error

	// Get retrieves a blob by digest, returning a streaming reader.
	Get(ctx context.Context, d digest.Digest) (io.ReadCloser, error)

	// Exists checks if a blob is present.
	Exists(ctx context.Context, d digest.Digest) (bool, error)

	// Writer returns a WriteCloser that streams into the blob store
	// and verifies the digest on Close.
	Writer(ctx context.Context, expected digest.Digest) (io.WriteCloser, error)
}

// BlobStoreFS implements BlobStore on the local filesystem.
type BlobStoreFS struct {
	basePath string
}

// NewBlobStoreFS creates a filesystem-backed blob store rooted at basePath.
func NewBlobStoreFS(basePath string) (*BlobStoreFS, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, err
	}
	return &BlobStoreFS{basePath: basePath}, nil
}

func (fs *BlobStoreFS) blobPath(d digest.Digest) (string, error) {
	hex := d.Hex()
	if len(hex) < 2 {
		return "", fmt.Errorf("invalid digest: %q", d)
	}
	dir := filepath.Join(fs.basePath, hex[:2])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, hex[2:]), nil
}

func (fs *BlobStoreFS) Put(ctx context.Context, d digest.Digest, r io.Reader) error {
	p, err := fs.blobPath(d)
	if err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Warnf("failed to close file: %v", cerr)
		}
	}()
	_, err = io.Copy(f, r)
	return err
}

func (fs *BlobStoreFS) Get(ctx context.Context, d digest.Digest) (io.ReadCloser, error) {
	p, err := fs.blobPath(d)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (fs *BlobStoreFS) Exists(ctx context.Context, d digest.Digest) (bool, error) {
	p, err := fs.blobPath(d)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

type verifyingWriter struct {
	f      *os.File
	dig    digest.Digester
	expect digest.Digest
}

func (vw *verifyingWriter) Write(p []byte) (int, error) {
	_, _ = vw.dig.Hash().Write(p)
	return vw.f.Write(p)
}

func (vw *verifyingWriter) Close() error {
	if err := vw.f.Close(); err != nil {
		return err
	}
	if got := vw.dig.Digest(); got != vw.expect {
		return fmt.Errorf("digest mismatch: got %s, want %s", got, vw.expect)
	}
	return nil
}

func (fs *BlobStoreFS) Writer(ctx context.Context, expected digest.Digest) (io.WriteCloser, error) {
	p, err := fs.blobPath(expected)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	return &verifyingWriter{
		f:      f,
		dig:    digest.Canonical.Digester(),
		expect: expected,
	}, nil
}

func (s *BlobStoreFS) WriterAtomic(ctx context.Context, dgst digest.Digest) (io.WriteCloser, error) {
	tmpPath := filepath.Join(s.basePath, dgst.Encoded()+".partial")
	finalPath := filepath.Join(s.basePath, "sha256", dgst.Encoded())

	f, err := os.Create(tmpPath)
	if err != nil {
		return nil, err
	}

	return &atomicWriter{
		File:      f,
		tmpPath:   tmpPath,
		finalPath: finalPath,
		expected:  dgst,
	}, nil
}

type atomicWriter struct {
	*os.File
	tmpPath, finalPath string
	expected           digest.Digest
	h                  hash.Hash
}

func (w *atomicWriter) Write(p []byte) (int, error) {
	if w.h == nil {
		w.h = sha256.New()
	}
	w.h.Write(p)
	return w.File.Write(p)
}

func (w *atomicWriter) Close() error {
	err := w.File.Close()
	if err != nil {
		os.Remove(w.tmpPath)
		return err
	}
	got := digest.NewDigest(digest.SHA256, w.h)
	if got != w.expected {
		os.Remove(w.tmpPath)
		return fmt.Errorf("digest mismatch: got %s, want %s", got, w.expected)
	}
	return os.Rename(w.tmpPath, w.finalPath)
}
