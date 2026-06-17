package application

import "io"

type UploadInput struct {
	WorkspaceID string
	UserID      string
	Filename    string
	File        io.Reader
}

type UploadedEvent struct {
	AudioID        string `json:"audio_id"`
	WorkspaceID    string `json:"workspace_id"`
	UploaderUserID string `json:"uploader_user_id,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
}

type AnonymousUploadInput struct {
	UserID   string
	Filename string
	File     io.Reader
}

type AnonymousSpeakerPayload struct {
	Label       string        `json:"label"`
	FragmentKey string        `json:"fragment_key"`
	Segments    []SegmentJSON `json:"segments"`
}

type SegmentJSON struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type AnonymousDiarizedEvent struct {
	SessionID   string                    `json:"session_id"`
	AudioID     string                    `json:"audio_id"`
	OwnerUserID string                    `json:"owner_user_id"`
	Speakers    []AnonymousSpeakerPayload `json:"speakers"`
}
