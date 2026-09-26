package mods

const modDBBaseURL = "https://mods.vintagestory.at"

type InstallInfoResult struct {
	Name               string `json:"-"`
	FileName           string `json:"fileName"`
	FileURL            string `json:"fileurl"`
	RecommendedUpgrade string `json:"recommendedUpgrade,omitempty"`
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

func downloadURL(fileURL string) string {
	return modDBBaseURL + fileURL
}

func (m ModInfo) QueryID() string {
	return m.ModID + "@" + m.Version
}
