package grpcserver

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"audio-ingest-service/internal/application"

	audiopb "audio-ingest-service/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AudioIngestServiceServer struct {
	audiopb.UnimplementedAudioIngestServiceServer

	storage       application.Storage
	accessChecker application.WorkspaceAccessChecker
}

func NewAudioIngestServiceServer(storage application.Storage, accessChecker application.WorkspaceAccessChecker) *AudioIngestServiceServer {
	return &AudioIngestServiceServer{
		storage:       storage,
		accessChecker: accessChecker,
	}
}

func (s *AudioIngestServiceServer) GetAudio(req *audiopb.GetAudioRequest, stream audiopb.AudioIngestService_GetAudioServer) error {
	if s.storage == nil || s.accessChecker == nil {
		return status.Error(codes.Internal, "grpc audio server is not configured")
	}
	if req == nil {
		return status.Error(codes.InvalidArgument, "request is required")
	}

	workspaceID := strings.TrimSpace(req.GetWorkspaceId())
	anonymous := workspaceID == ""

	if !anonymous {
		if _, err := uuid.Parse(workspaceID); err != nil {
			return status.Error(codes.InvalidArgument, "workspace_id must be a valid UUID")
		}
	}

	audioID := strings.TrimSpace(req.GetAudioId())
	if _, err := uuid.Parse(audioID); err != nil {
		return status.Error(codes.InvalidArgument, "audio_id must be a valid UUID")
	}

	uploaderUserID := strings.TrimSpace(req.GetUploaderUserId())
	if uploaderUserID == "" {
		return status.Error(codes.InvalidArgument, "uploader_user_id is required")
	}

	log := slog.Default().With("audio_id", audioID, "workspace_id", workspaceID, "anonymous", anonymous)
	log.Info("gRPC GetAudio request received")

	if !anonymous {
		allowed, err := s.accessChecker.CanUpload(stream.Context(), workspaceID, uploaderUserID)
		if err != nil {
			log.Error("workspace access check failed", "error", err)
			return status.Errorf(codes.Internal, "workspace access check: %v", err)
		}
		if !allowed {
			log.Warn("gRPC GetAudio access denied", "uploader_user_id", uploaderUserID)
			return status.Error(codes.PermissionDenied, "workspace access denied")
		}
	}

	objectKey := buildGRPCAudioObjectKey(workspaceID, audioID, anonymous)
	reader, err := s.storage.Get(stream.Context(), objectKey)
	if err != nil {
		log.Error("failed to load audio from storage", "object_key", objectKey, "error", err)
		return status.Errorf(codes.NotFound, "load audio: %v", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	const chunkSize = 64 * 1024
	buffer := make([]byte, chunkSize)
	chunks := 0

	for {
		readBytes, readErr := reader.Read(buffer)
		if readBytes > 0 {
			content := make([]byte, readBytes)
			copy(content, buffer[:readBytes])
			if err := stream.Send(&audiopb.AudioChunk{
				AudioId:     audioID,
				WorkspaceId: workspaceID,
				Content:     content,
			}); err != nil {
				log.Error("failed to send audio chunk", "chunk", chunks, "error", err)
				return status.Errorf(codes.Internal, "send audio chunk: %v", err)
			}
			chunks++
			log.Debug("sent audio chunk", "chunk", chunks, "bytes", readBytes)
		}

		if readErr == io.EOF {
			log.Info("gRPC GetAudio stream completed", "chunks_sent", chunks)
			return nil
		}
		if readErr != nil {
			log.Error("failed to read audio object", "error", readErr)
			return status.Errorf(codes.Internal, "read audio object: %v", readErr)
		}
	}
}

func buildGRPCAudioObjectKey(workspaceID, audioID string, anonymous bool) string {
	if anonymous {
		return fmt.Sprintf("anonymous/%s.mp3", audioID)
	}
	return fmt.Sprintf("%s/%s.mp3", workspaceID, audioID)
}
