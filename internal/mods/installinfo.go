package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const installInfoURL = "https://mods.vintagestory.at/api/v2/mods/install-information"

func FetchInstallInfo(ctx context.Context, queryIDs []string, gameVersion string) ([]InstallInfoResult, error) {
	if len(queryIDs) == 0 {
		return nil, nil
	}

	params := url.Values{}
	params.Set("ids", strings.Join(queryIDs, ","))
	if gameVersion != "" {
		params.Set("gv", gameVersion)
	}

	reqURL := installInfoURL + "?" + params.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching install info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var parsed installInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	results := make([]InstallInfoResult, 0, len(parsed.Data))
	for modid, info := range parsed.Data {
		info.Name = modid
		results = append(results, info)
	}
	return results, nil
}