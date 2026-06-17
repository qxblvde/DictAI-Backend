package grpcserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"

	"audio-ingest-service/internal/application"

	audiopb "audio-ingest-service/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type grpcTestStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
	getErr  error
}

func (s *grpcTestStorage) Put(_ context.Context, objectKey string, body io.Reader) error {
	payload, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.objects == nil {
		s.objects = map[string][]byte{}
	}
	s.objects[objectKey] = payload
	return nil
}

func (s *grpcTestStorage) Get(_ context.Context, objectKey string) (io.ReadCloser, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}

	s.mu.Lock()
	payload, ok := s.objects[objectKey]
	s.mu.Unlock()
	if !ok {
		return nil, errors.New("object not found")
	}

	body := make([]byte, len(payload))
	copy(body, payload)
	return io.NopCloser(bytes.NewReader(body)), nil
}

func (s *grpcTestStorage) Delete(_ context.Context, objectKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, objectKey)
	return nil
}

type grpcTestAccessChecker struct {
	mu              sync.Mutex
	allowed         bool
	checkErr        error
	calls           int
	lastWorkspaceID string
	lastUserID      string
}

func (c *grpcTestAccessChecker) CanUpload(_ context.Context, workspaceID, userID string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls++
	c.lastWorkspaceID = workspaceID
	c.lastUserID = userID

	if c.checkErr != nil {
		return false, c.checkErr
	}
	return c.allowed, nil
}

func TestGetAudio_Success(t *testing.T) {
	workspaceID := uuid.NewString()
	audioID := uuid.NewString()
	uploaderUserID := "user-1"
	payload := []byte("hello world")

	storage := &grpcTestStorage{
		objects: map[string][]byte{
			buildGRPCAudioObjectKey(workspaceID, audioID): payload,
		},
	}
	checker := &grpcTestAccessChecker{allowed: true}
	client, cleanup := newAudioIngestClient(t, storage, checker)
	defer cleanup()

	stream, err := client.GetAudio(context.Background(), &audiopb.GetAudioRequest{
		AudioId:        audioID,
		WorkspaceId:    workspaceID,
		UploaderUserId: uploaderUserID,
	})
	require.NoError(t, err)

	var got bytes.Buffer
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		assert.Equal(t, audioID, chunk.GetAudioId())
		assert.Equal(t, workspaceID, chunk.GetWorkspaceId())
		_, err = got.Write(chunk.GetContent())
		require.NoError(t, err)
	}

	assert.Equal(t, string(payload), got.String())

	checker.mu.Lock()
	assert.Equal(t, 1, checker.calls)
	assert.Equal(t, workspaceID, checker.lastWorkspaceID)
	assert.Equal(t, uploaderUserID, checker.lastUserID)
	checker.mu.Unlock()
}

func TestGetAudio_InvalidWorkspaceID(t *testing.T) {
	storage := &grpcTestStorage{}
	checker := &grpcTestAccessChecker{allowed: true}
	client, cleanup := newAudioIngestClient(t, storage, checker)
	defer cleanup()

	stream, err := client.GetAudio(context.Background(), &audiopb.GetAudioRequest{
		AudioId:        uuid.NewString(),
		WorkspaceId:    "bad-workspace",
		UploaderUserId: "user-1",
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	checker.mu.Lock()
	assert.Equal(t, 0, checker.calls)
	checker.mu.Unlock()
}

func TestGetAudio_ForbiddenWhenWorkspaceAccessDenied(t *testing.T) {
	storage := &grpcTestStorage{}
	checker := &grpcTestAccessChecker{allowed: false}
	client, cleanup := newAudioIngestClient(t, storage, checker)
	defer cleanup()

	stream, err := client.GetAudio(context.Background(), &audiopb.GetAudioRequest{
		AudioId:        uuid.NewString(),
		WorkspaceId:    uuid.NewString(),
		UploaderUserId: "user-1",
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	checker.mu.Lock()
	assert.Equal(t, 1, checker.calls)
	checker.mu.Unlock()
}

func TestGetAudio_StorageErrorReturnsNotFound(t *testing.T) {
	storage := &grpcTestStorage{getErr: errors.New("minio is down")}
	checker := &grpcTestAccessChecker{allowed: true}
	client, cleanup := newAudioIngestClient(t, storage, checker)
	defer cleanup()

	stream, err := client.GetAudio(context.Background(), &audiopb.GetAudioRequest{
		AudioId:        uuid.NewString(),
		WorkspaceId:    uuid.NewString(),
		UploaderUserId: "user-1",
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetAudio_AccessCheckerError(t *testing.T) {
	storage := &grpcTestStorage{}
	checker := &grpcTestAccessChecker{allowed: true, checkErr: errors.New("workspace service unavailable")}
	client, cleanup := newAudioIngestClient(t, storage, checker)
	defer cleanup()

	stream, err := client.GetAudio(context.Background(), &audiopb.GetAudioRequest{
		AudioId:        uuid.NewString(),
		WorkspaceId:    uuid.NewString(),
		UploaderUserId: "user-1",
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func newAudioIngestClient(t *testing.T, storage application.Storage, checker application.WorkspaceAccessChecker) (audiopb.AudioIngestServiceClient, func()) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	audiopb.RegisterAudioIngestServiceServer(server, NewAudioIngestServiceServer(storage, checker))

	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := audiopb.NewAudioIngestServiceClient(conn)

	cleanup := func() {
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
	}

	return client, cleanup
}
