package domain

import "time"

type Result struct {
	ResultID      string
	AudioID       string
	WorkspaceID   string
	WorkspaceName string
	UploadUserID  string
	SummaryKey    string
	TranscriptKey string
	Status        string
	CreatedAt     time.Time
}
