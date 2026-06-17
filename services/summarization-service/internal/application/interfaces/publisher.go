package interfaces

type Publisher interface {
	Publish(audioID, workspaceID, uploadUserID, summaryURL, transcriptURL string) error
}
