package versions

const VERSIONS_URL_STABLE = "https://api.vintagestory.at/stable.json"
const VERSIONS_URL_UNSTABLE = "https://api.vintagestory.at/unstable.json"

type PlatformFile struct {
	Filename string `json:"filename"`
	FileSize string `json:"filesize"`
	MD5      string `json:"md5"`
	URLs     struct {
		CDN   string `json:"cdn"`
		LOCAL string `json:"local"`
	} `json:"urls"`
	Latest int `json:"latest"`
}

type versionsResponse map[string]map[string]PlatformFile

type Release struct {
	Version      string `json:"version"`
	Channel      string `json:"channel"`
	Platform     string `json:"platform"`
	Filename     string `json:"filename"`
	DownloadURL  string `json:"download_url"`
	Checksum     string `json:"checksum"`
	ChecksumAlgo string `json:"checksum_algo"`
	DownloadSize int64  `json:"-"`
	FileSizeHuman string `json:"file_size"`
	Latest       bool   `json:"latest"`
}