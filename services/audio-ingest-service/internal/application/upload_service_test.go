package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	domainerrors "audio-ingest-service/internal/domain/errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStorage struct {
	putCalls    int
	deleteCalls int
	lastKey     string
	lastBody    []byte
	putErr      error
	deleteErr   error
}

func (f *fakeStorage) Put(_ context.Context, objectKey string, body io.Reader) error {
	f.putCalls++
	f.lastKey = objectKey

	if f.putErr != nil {
		return f.putErr
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.lastBody = data
	return nil
}

func (f *fakeStorage) Get(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(string(f.lastBody))), nil
}

func (f *fakeStorage) Delete(_ context.Context, objectKey string) error {
	f.deleteCalls++
	f.lastKey = objectKey
	return f.deleteErr
}

type fakePublisher struct {
	publishCalls int
	lastEvent    UploadedEvent
	publishErr   error
}

func (f *fakePublisher) PublishAudioUploaded(_ context.Context, event UploadedEvent) error {
	f.publishCalls++
	f.lastEvent = event
	return f.publishErr
}

type fakeAccessChecker struct {
	calls      int
	allowed    bool
	checkErr   error
	lastUserID string
}

func (f *fakeAccessChecker) CanUpload(_ context.Context, _ string, userID string) (bool, error) {
	f.calls++
	f.lastUserID = userID
	if f.checkErr != nil {
		return false, f.checkErr
	}
	return f.allowed, nil
}

func TestUpload_Success(t *testing.T) {
	storage := &fakeStorage{}
	publisher := &fakePublisher{}
	checker := &fakeAccessChecker{allowed: true}

	service := NewUploadService(storage, publisher, checker)
	workspaceID := uuid.NewString()

	audioID, err := service.Upload(context.Background(), UploadInput{
		WorkspaceID: workspaceID,
		UserID:      "user-1",
		Filename:    "meeting.MP3",
		File:        strings.NewReader("audio-bytes"),
	})

	require.NoError(t, err)
	require.NotEmpty(t, audioID)
	_, parseErr := uuid.Parse(audioID)
	require.NoError(t, parseErr)

	assert.Equal(t, 1, checker.calls)
	assert.Equal(t, 1, storage.putCalls)
	assert.Equal(t, 1, publisher.publishCalls)
	assert.Equal(t, "audio-bytes", string(storage.lastBody))
	assert.Contains(t, storage.lastKey, workspaceID+"/")
	assert.Contains(t, storage.lastKey, ".mp3")
	assert.Equal(t, "user-1", publisher.lastEvent.UploaderUserID)
}

func TestUpload_Forbidden(t *testing.T) {
	storage := &fakeStorage{}
	publisher := &fakePublisher{}
	checker := &fakeAccessChecker{allowed: false}

	service := NewUploadService(storage, publisher, checker)

	_, err := service.Upload(context.Background(), UploadInput{
		WorkspaceID: uuid.NewString(),
		UserID:      "user-1",
		File:        strings.NewReader("audio-bytes"),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domainerrors.ErrForbidden)
	assert.Equal(t, 1, checker.calls)
	assert.Equal(t, 0, storage.putCalls)
	assert.Equal(t, 0, publisher.publishCalls)
}

func TestUpload_RollbackOnPublishFailure(t *testing.T) {
	storage := &fakeStorage{}
	publisher := &fakePublisher{publishErr: errors.New("nats down")}
	checker := &fakeAccessChecker{allowed: true}

	service := NewUploadService(storage, publisher, checker)

	_, err := service.Upload(context.Background(), UploadInput{
		WorkspaceID: uuid.NewString(),
		UserID:      "user-1",
		Filename:    "meeting.wav",
		File:        strings.NewReader("audio-bytes"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish audio.uploaded")
	assert.Equal(t, 1, storage.putCalls)
	assert.Equal(t, 1, storage.deleteCalls)
}

func TestUpload_InvalidWorkspaceID(t *testing.T) {
	storage := &fakeStorage{}
	publisher := &fakePublisher{}
	checker := &fakeAccessChecker{allowed: true}

	service := NewUploadService(storage, publisher, checker)

	_, err := service.Upload(context.Background(), UploadInput{
		WorkspaceID: "not-a-uuid",
		UserID:      "user-1",
		File:        strings.NewReader("audio-bytes"),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidInput)
	assert.Equal(t, 0, checker.calls)
	assert.Equal(t, 0, storage.putCalls)
	assert.Equal(t, 0, publisher.publishCalls)
}
