package interfaces

import "context"

type Segment struct {
	ParticipantID   string  `json:"participantId"`
	ParticipantName string  `json:"participantName,omitempty"`
	Start           float64 `json:"start"`
	End             float64 `json:"end"`
	Text            string  `json:"text"`
}

type MeetingMeta struct {
	WorkspaceName string `json:"workspaceName,omitempty"`
	UploaderName  string `json:"uploaderName,omitempty"`
	Date          string `json:"date,omitempty"`
}

type SummaryResult struct {
	SummaryFragment    string
	TranscriptFragment string
}

type Summarizer interface {
	Summarize(ctx context.Context, segments []Segment) (*SummaryResult, error)
	SummarizeWithMeta(ctx context.Context, segments []Segment, meta MeetingMeta) (*SummaryResult, error)
}
