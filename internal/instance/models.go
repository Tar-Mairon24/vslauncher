package instance

import "time"

type Instance struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Channel     string    `json:"channel"`
	Platform    string    `json:"platform"`
	Path        string    `json:"path"`
	DataPath    string    `json:"data_path"`
	InstalledAt time.Time `json:"installed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type registryEntry struct {
	Name 	  string    `json:"name"`
	Path	  string    `json:"path"`
}
