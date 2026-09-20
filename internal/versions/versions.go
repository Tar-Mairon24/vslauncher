package versions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

func FetchReleases(ctx context.Context, url, channel string) ([]Release, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching %s: %s", url, resp.Status)
	}

	var raw versionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding versions response: %w", err)
	}

	var releases []Release
	for versionStr, platforms := range raw {
		for platform, pf := range platforms {
			size, err := parseFileSize(pf.FileSize)
			if err != nil {
				fmt.Printf("skipping %s/%s: %v", versionStr, platform, err)
				continue
			}
			releases = append(releases, Release{
				Version:      versionStr,
				Channel:      channel,
				Platform:     platform,
				Filename:     pf.Filename,
				DownloadURL:  pf.URLs.CDN,
				Checksum:     pf.MD5,
				ChecksumAlgo: "md5",
				DownloadSize: size,
				FileSizeHuman: pf.FileSize,
				Latest:       pf.Latest == 1,
			})
		}
	}
	return releases, nil
}

func parseFileSize(s string) (int64, error) {
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0, fmt.Errorf("unexpected file size format: %q", s)
	}
	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parsing size number %q: %w", parts[0], err)
	}

	var multiplier float64
	switch strings.ToUpper(parts[1]) {
	case "KB":
		multiplier = 1 << 10
	case "MB":
		multiplier = 1 << 20
	case "GB":
		multiplier = 1 << 30
	default:
		return 0, fmt.Errorf("unknown size unit: %q", parts[1])
	}

	return int64(value * multiplier), nil
}

func SortReleasesByVersion(releases []Release, operation string) {
	desc := operation == "desc"

	for _, r := range releases {
		if !semver.IsValid("v" + r.Version) {
			fmt.Printf("warning: %q is not valid semver, sort order may be wrong", r.Version)
		}
	}

	sort.Slice(releases, func(i, j int) bool {
		vi, vj := "v"+releases[i].Version, "v"+releases[j].Version
		cmp := semver.Compare(vi, vj)
		if cmp != 0 {
			if desc {
				return cmp > 0
			}
			return cmp < 0
		}
		if desc {
			return releases[i].Platform > releases[j].Platform
		}
		return releases[i].Platform < releases[j].Platform
	})
}

func FilterReleasesByPlatform(releases []Release, platform string) []Release {
	if platform == "" {
		return releases
	}

	var filtered []Release
	for _, r := range releases {
		if strings.EqualFold(r.Platform, platform) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func FilterReleasesByChannel(releases []Release, channel string) []Release {
	if channel == "" {
		return releases
	}

	var filtered []Release
	for _, r := range releases {
		if strings.EqualFold(r.Channel, channel) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func FindRelease(releases []Release, version string, platform string) (Release, error) {
	for _, r := range releases {
		if r.Version == version && strings.EqualFold(r.Platform, platform) {
			return r, nil
		}
	}
	return Release{}, fmt.Errorf("release not found for version %q and platform %q", version, platform)
}