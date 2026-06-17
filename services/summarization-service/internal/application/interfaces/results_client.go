package interfaces

import "context"

type UploadInput struct {
	AudioID       string
	WorkspaceID   string
	UploadUserID  string
	SummaryPDF    []byte
	TranscriptPDF []byte
}

type UploadOutput struct {
	ResultID      string
	SummaryURL    string
	TranscriptURL string
}

type ResultsClient interface {
	Upload(ctx context.Context, in UploadInput) (*UploadOutput, error)
}
