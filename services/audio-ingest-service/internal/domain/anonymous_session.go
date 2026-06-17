package domain

import "time"

type SessionStatus string

const (
	SessionStatusProcessing SessionStatus = "processing"
	SessionStatusReady      SessionStatus = "ready"
	SessionStatusConfirmed  SessionStatus = "confirmed"
	SessionStatusExpired    SessionStatus = "expired"
)

type AnonymousSession struct {
	SessionID   string
	AudioID     string
	OwnerUserID string
	Status      SessionStatus
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type SegmentInfo struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type AnonymousSpeaker struct {
	SpeakerID   string
	SessionID   string
	Label       string
	FragmentKey string
	Segments    []SegmentInfo
	CreatedAt   time.Time
}
