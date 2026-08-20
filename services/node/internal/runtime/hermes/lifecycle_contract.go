package hermes

const (
	lifecycleOfficialVersion  = "0.20.2"
	lifecycleStatusRunning    = "Gateway process running"
	lifecycleStatusStopped    = "No gateway process detected"
	lifecycleManualRunning    = "Gateway is running (PID:"
	lifecycleManualStopped    = "Gateway is not running"
	lifecycleTaskInstalled    = "Scheduled Task registered"
	lifecycleStartupInstalled = "Windows login item installed"
	lifecycleServiceMissing   = "Gateway service not installed"
)

func lifecycleProfileArgs(nativeID string, tail ...string) []string {
	args := make([]string, 0, len(tail)+2)
	if nativeID != "default" {
		args = append(args, "--profile", nativeID)
	}
	return append(args, tail...)
}

func lifecycleStatusArgs(nativeID string) []string {
	return lifecycleProfileArgs(nativeID, "gateway", "status")
}

func lifecycleStartArgs(nativeID string, loginItemPresent bool) []string {
	if loginItemPresent {
		return lifecycleProfileArgs(nativeID, "gateway", "start")
	}
	return lifecycleProfileArgs(nativeID, "gateway", "install", "--no-start-on-login", "--start-now")
}

func lifecycleStopArgs(nativeID string) []string {
	return lifecycleProfileArgs(nativeID, "gateway", "stop")
}

func lifecycleRestartArgs(nativeID string) []string {
	return lifecycleProfileArgs(nativeID, "gateway", "restart")
}
