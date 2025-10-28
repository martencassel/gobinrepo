package blobs

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"

	digest "github.com/opencontainers/go-digest"
	assert "github.com/stretchr/testify/require"
)

func getRandomBlobData(size int) []byte {
	randData := make([]byte, size)
	_, err := rand.Read(randData)
	if err != nil {
		panic(err)
	}
	data := make([]byte, size)
	copy(data, randData)
	return data
}

func TestBlobs(t *testing.T) {
	bfs, err := NewBlobStoreFS("/tmp/blobstore_test")
	assert.NoError(t, err, "Failed to create BlobStoreFS")
	blobData := getRandomBlobData(1024 * 10) // 10KB blob
	d := digest.FromBytes(blobData)
	err = bfs.Put(context.Background(), d, bytes.NewReader(blobData))
	assert.NoError(t, err, "Failed to put blob")
	exists, err := bfs.Exists(context.Background(), d)
	assert.NoError(t, err, "Failed to check blob existence")
	assert.True(t, exists, "Blob should exist")
	reader, err := bfs.Get(context.Background(), d)
	assert.NoError(t, err, "Failed to get blob")
	retrievedData := make([]byte, len(blobData))
	_, err = reader.Read(retrievedData)
	assert.NoError(t, err, "Failed to read blob data")
	assert.Equal(t, blobData, retrievedData, "Retrieved blob data does not match original")
}
