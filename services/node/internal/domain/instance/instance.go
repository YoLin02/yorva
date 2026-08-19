package instance

import "time"

type Availability string

const (
	Available Availability = "AVAILABLE"
	Missing   Availability = "MISSING"
	Unknown   Availability = "UNKNOWN"
)

type Instance struct {
	ID                    string
	RuntimeInstallationID string
	NativeID              string
	Name                  string
	Default               bool
	Protected             bool
	Availability          Availability
	LastSyncedAt          *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func NewID(random string) string {
	return "inst_" + random
}

func (i Instance) LifecycleSupported() bool {
	return false
}
