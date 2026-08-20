package channel

import "time"

type Type string

const (
	Weixin Type = "weixin"
	WeCom  Type = "wecom"
)

type State string

const (
	NotConfigured State = "NOT_CONFIGURED"
	Connecting    State = "CONNECTING"
	Connected     State = "CONNECTED"
	Disconnected  State = "DISCONNECTED"
	Failed        State = "FAILED"
	Unknown       State = "UNKNOWN"
)

type Binding struct {
	ID            string
	InstanceID    string
	Type          Type
	State         State
	AccountLabel  string
	ExternalID    string
	LastCheckedAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func ValidType(value Type) bool {
	return value == Weixin || value == WeCom
}

func ValidState(value State) bool {
	switch value {
	case NotConfigured, Connecting, Connected, Disconnected, Failed, Unknown:
		return true
	default:
		return false
	}
}
