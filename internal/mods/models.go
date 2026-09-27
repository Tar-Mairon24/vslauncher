package mods

import "github.com/Tar-Mairon24/vslauncher/internal/download"

const modDBBaseURL = "https://mods.vintagestory.at"

type InstallInfoResult struct {
	Name               string `json:"-"`
	FileName           string `json:"fileName"`
	FileURL            string `json:"fileUrl"`
	RecommendedUpgrade string `json:"recommendedUpgrade,omitempty"`
	ErrorCode          int    `json:"errorCode,omitempty"`
	RetractionReason   string `json:"retractionReason,omitempty"`
}

type installInfoResponse struct {
	Data map[string]InstallInfoResult `json:"data"`
}

type ModInfo struct {
	Type             string            `json:"type"`
	ModID            string            `json:"modid"`
	Name             string            `json:"name"`
	Authors          []string          `json:"authors"`
	Contributors     []string          `json:"contributors"`
	Version          string            `json:"version"`
	Dependencies     map[string]string `json:"dependencies"`
	Side             string            `json:"side"`
	RequiredOnClient bool              `json:"requiredonclient"`
	RequiredOnServer bool              `json:"requiredonserver"`
}

type InstalledMod struct {
	Info    ModInfo
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

type clientSettings struct {
	StringListSettings struct {
		DisabledMods []string `json:"disabledMods"`
	} `json:"stringListSettings"`
}

type UpdateResult struct {
	ModID      string
	OldVersion string
	NewVersion string
	Error      string
}

type UpdateModParams struct {
	Installed      []InstalledMod
	ModsDir        string
	DataPath       string
	GameVersion    string
	TargetModIDs   []string
	ExcludedModIDs []string
	InstName       string
	BackupDir      string
	MaxBackups     int
	NewProgress    download.MultiProgressFactory
}

func (r InstallInfoResult) Available() bool {
	return r.ErrorCode == 0
}

func downloadURL(fileURL string) string {
	return modDBBaseURL + fileURL
}

func (m ModInfo) QueryID() string {
	return m.ModID + "@" + m.Version
}
