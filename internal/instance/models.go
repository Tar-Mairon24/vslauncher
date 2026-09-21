package instance

import (
	"text/template"
	"time"
)

type Instance struct {
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Channel     string     `json:"channel"`
	Platform    string     `json:"platform"`
	Path        string     `json:"path"`
	DataPath    string     `json:"data_path"`
	DesktopPath string     `json:"desktop_path"`
	IconPath    string     `json:"icon_path"`
	InstalledAt time.Time  `json:"installed_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type registryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type desktopData struct {
	*Instance
	BinaryPath string
}

var desktopLinuxTemplate = template.Must(template.New("desktopLinux").Parse(`
#!/usr/bin/xdg-open
[Desktop Entry]
Name= Vintage Story - {{.Name}}
Name[de]= VSL Instance - {{.Name}}
GenericName=Sandbox Game
GenericName[de]=Sandbox Spiel
Comment=VSL - Vintage Story Instance {{.Name}} {{.Version}} ({{.Channel}})
Terminal=false
Type=Application
StartupNotify=false
Categories=Game;
Path={{.Path}}
Exec={{.BinaryPath}} --dataPath {{.DataPath}}
Icon={{.IconPath}}
`))
