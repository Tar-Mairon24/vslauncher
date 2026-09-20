package instance

import "time"

type Instance struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Channel     string    `json:"channel"`
	Platform    string    `json:"platform"`
	Path        string    `json:"path"`
	InstalledAt time.Time `json:"installed_at"`
}

type registryEntry struct {
	Name 	  string    `json:"name"`
	Path	  string    `json:"path"`
}
