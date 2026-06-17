package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMinIOClient struct {
	bucketExists bool
	bucketErr    error
	makeErr      error
	putErr       error
	removeErr    error

	makeCalls   int
	putCalls    int
	removeCalls int

	lastBucket string
	lastKey    string
	lastBody   []byte
}

func (f *fakeMinIOClient) PutObject(_ context.Context, bucketName, objectName string, reader io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	f.putCalls++
	f.lastBucket = bucketName
	f.lastKey = objectName

	if f.putErr != nil {
		return minio.UploadInfo{}, f.putErr
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	f.lastBody = body
	return minio.UploadInfo{}, nil
}

func (f *fakeMinIOClient) GetObject(_ context.Context, bucketName, objectName string, _ minio.GetObjectOptions) (*minio.Object, error) {
	f.lastBucket = bucketName
	f.lastKey = objectName
	return nil, errors.New("not implemented")
}

func (f *fakeMinIOClient) RemoveObject(_ context.Context, bucketName, objectName string, _ minio.RemoveObjectOptions) error {
	f.removeCalls++
	f.lastBucket = bucketName
	f.lastKey = objectName
	return f.removeErr
}

func (f *fakeMinIOClient) BucketExists(_ context.Context, bucketName string) (bool, error) {
	f.lastBucket = bucketName
	if f.bucketErr != nil {
		return false, f.bucketErr
	}
	return f.bucketExists, nil
}

func (f *fakeMinIOClient) MakeBucket(_ context.Context, bucketName string, _ minio.MakeBucketOptions) error {
	f.makeCalls++
	f.lastBucket = bucketName
	return f.makeErr
}

func TestNewMinIOStorageWithClient_CreatesBucketWhenMissing(t *testing.T) {
	client := &fakeMinIOClient{bucketExists: false}

	storage, err := newMinIOStorageWithClient(context.Background(), client, "audio")
	require.NoError(t, err)
	require.NotNil(t, storage)
	assert.Equal(t, 1, client.makeCalls)
}

func TestNewMinIOStorageWithClient_SkipsCreateWhenBucketExists(t *testing.T) {
	client := &fakeMinIOClient{bucketExists: true}

	storage, err := newMinIOStorageWithClient(context.Background(), client, "audio")
	require.NoError(t, err)
	require.NotNil(t, storage)
	assert.Equal(t, 0, client.makeCalls)
}

func TestMinIOStorage_PutAndDelete(t *testing.T) {
	client := &fakeMinIOClient{bucketExists: true}
	storage, err := newMinIOStorageWithClient(context.Background(), client, "audio")
	require.NoError(t, err)

	err = storage.Put(context.Background(), "workspace/audio.bin", strings.NewReader("audio-bytes"))
	require.NoError(t, err)
	assert.Equal(t, 1, client.putCalls)
	assert.Equal(t, "audio", client.lastBucket)
	assert.Equal(t, "workspace/audio.bin", client.lastKey)
	assert.Equal(t, "audio-bytes", string(client.lastBody))

	err = storage.Delete(context.Background(), "workspace/audio.bin")
	require.NoError(t, err)
	assert.Equal(t, 1, client.removeCalls)
}

func TestMinIOStorage_PutError(t *testing.T) {
	client := &fakeMinIOClient{bucketExists: true, putErr: errors.New("put failed")}
	storage, err := newMinIOStorageWithClient(context.Background(), client, "audio")
	require.NoError(t, err)

	err = storage.Put(context.Background(), "workspace/audio.bin", strings.NewReader("audio-bytes"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "put object to minio")
}
