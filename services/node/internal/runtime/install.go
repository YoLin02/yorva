package runtime

import "context"

type InstallRequest struct {
	RuntimeKind Kind
}

type Installation struct {
	RuntimeKind  Kind
	Path         string
	Version      string
	SupportState DiscoveryState
}

type ProgressUpdate struct {
	Stage        string
	WarningCodes []string
}

type ProgressSink interface {
	Report(ProgressUpdate)
}

type Installer interface {
	Install(ctx context.Context, req InstallRequest, progress ProgressSink) (Installation, error)
}
