package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Microservices/services/notification-service/internal/infrastructure/auth"
	"github.com/Microservices/services/notification-service/internal/infrastructure/email"
	"github.com/Microservices/services/notification-service/internal/infrastructure/workspace"
)

type EmailSender interface {
	Send(to, subject, body string) error
}

type WorkspaceClient interface {
	GetParticipants(ctx context.Context, workspaceID, userID string) ([]workspace.Participant, error)
	GetWorkspace(ctx context.Context, workspaceID, userID string) (*workspace.WorkspaceInfo, error)
}

type AuthClient interface {
	GetUserEmail(ctx context.Context, userID string) (string, error)
}

type NotificationService struct {
	sender    EmailSender
	workspace WorkspaceClient
	auth      AuthClient
}

func New(sender *email.SMTPSender, wc *workspace.HTTPClient, ac *auth.HTTPClient) *NotificationService {
	return &NotificationService{sender: sender, workspace: wc, auth: ac}
}

func (s *NotificationService) Notify(ctx context.Context, audioID, workspaceID, uploadUserID, summaryURL, transcriptURL string) {
	log := slog.With("workspace_id", workspaceID, "audio_id", audioID)

	participants, err := s.workspace.GetParticipants(ctx, workspaceID, uploadUserID)
	if err != nil {
		log.Error("failed to fetch participants", "error", err)
		return
	}

	workspaceName := workspaceID
	if ws, wsErr := s.workspace.GetWorkspace(ctx, workspaceID, uploadUserID); wsErr == nil {
		workspaceName = ws.Name
	} else {
		log.Warn("failed to fetch workspace name", "error", wsErr)
	}

	var names []string
	for _, p := range participants {
		names = append(names, p.Name)
	}

	subject := fmt.Sprintf("Результаты встречи: %s", workspaceName)
	body := fmt.Sprintf(
		"Встреча: %s\nУчастники: %s\n\nРезультаты готовы:\nSummary: %s\nТранскрипт: %s",
		workspaceName,
		strings.Join(names, ", "),
		summaryURL,
		transcriptURL,
	)

	// Collect all recipient emails (deduplicated)
	emails := map[string]struct{}{}

	// Owner's account email
	if ownerEmail, authErr := s.auth.GetUserEmail(ctx, uploadUserID); authErr == nil && ownerEmail != "" {
		emails[ownerEmail] = struct{}{}
	} else if authErr != nil {
		log.Warn("failed to fetch owner email", "error", authErr)
	}

	// Participants' emails
	for _, p := range participants {
		if p.Email != "" {
			emails[p.Email] = struct{}{}
		} else {
			log.Warn("participant has no email, skipping", "name", p.Name)
		}
	}

	log.Info("sending notifications", "recipients", len(emails))
	for to := range emails {
		if err := s.sender.Send(to, subject, body); err != nil {
			log.Error("failed to send email", "to", to, "error", err)
		} else {
			log.Info("email sent", "to", to)
		}
	}
}
