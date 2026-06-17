package domain

import "time"

type Workspace struct {
	WorkspaceID string
	OwnerID     string
	Name        string
	CreatedAt   time.Time
}
