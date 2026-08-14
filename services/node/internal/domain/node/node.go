package node

import "time"

type Node struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Hostname     string    `json:"hostname"`
	Platform     string    `json:"platform"`
	Architecture string    `json:"architecture"`
	NodeVersion  string    `json:"nodeVersion"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type LocalMetadata struct {
	Name         string
	Hostname     string
	Platform     string
	Architecture string
	NodeVersion  string
}
