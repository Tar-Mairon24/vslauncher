package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Tar-Mairon24/vslauncher/internal/versions"
)

var channelURLs = map[string]string{
	"stable":   versions.VERSIONS_URL_STABLE,
	"unstable": versions.VERSIONS_URL_UNSTABLE,
}

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "Manage and inspect Vintage Story versions",
	Long:  "Manage and inspect Vintage Story versions, including listing available versions and filtering by channel, operation, and platform.",
	Aliases: []string{"ver", "v"},
}

var validPlatforms = map[string]bool{
	"windows": true, "windowsupdate": true, "linux": true,
	"mac-x64": true, "mac-arm64": true, "server": true,
}

func resolveReleases(channel, operation, platform string) ([]versions.Release, error) {
	if operation != "asc" && operation != "desc" {
		return nil, fmt.Errorf("unknown operation %q", operation)
	}
	if platform != "" && !validPlatforms[platform] {
		return nil, fmt.Errorf("unknown platform %q", platform)
	}

	var releases []versions.Release
	channelsToFetch := []string{"stable", "unstable"}
	if channel != "" {
		url, ok := channelURLs[channel]
		if !ok {
			return nil, fmt.Errorf("unknown channel %q", channel)
		}
		r, err := versions.FetchReleases(context.Background(), url, channel)
		if err != nil {
			return nil, fmt.Errorf("fetching %s releases: %w", channel, err)
		}
		releases = r
	} else {
		for _, ch := range channelsToFetch {
			r, err := versions.FetchReleases(context.Background(), channelURLs[ch], ch)
			if err != nil {
				return nil, fmt.Errorf("fetching %s releases: %w", ch, err)
			}
			releases = append(releases, r...)
		}
	}

	if platform != "" {
		releases = versions.FilterReleasesByPlatform(releases, platform)
	}
	versions.SortReleasesByVersion(releases, operation)
	return releases, nil
}

func listCmd() *cobra.Command {
	var channel, operation, platform string
	var verbose bool
	var head int
	cmd := &cobra.Command{
		Use:   "list [-c channel] [-o operation] [-p platform]",
		Short: "List available Vintage Story versions",
		Long:  "List available Vintage Story versions, optionally filtered by channel, operation, and platform.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var releases []versions.Release
			var err error
			if platform != "" {
				if !validPlatforms[platform] {
					return fmt.Errorf("unknown platform %q", platform)
				}
				releases, err = resolveReleases(channel, operation, platform)
				if err != nil {
					return err
				}
			} else {
				releases, err = resolveReleases(channel, operation, currentPlatform)
				if err != nil {
					return err
				}
			}

			if err != nil {
				return err
			}

			if head > 0 && head < len(releases) {
				releases = releases[:head]
			}
			
			if verbose {
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "VERSION\tCHANNEL\tPLATFORM\tSIZE\tLATEST")
			for _, r := range releases {
				latest := ""
				if r.Latest {
					latest = "yes"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					r.Version, r.Channel, r.Platform, r.FileSizeHuman, latest)
			}
			return w.Flush()
		}

		for _, r := range releases {
			fmt.Println(r.Version)
		}
		return nil
		},
	}
	cmd.Flags().StringVarP(&channel, "channel", "c", "", "release channel (stable or unstable)")
	cmd.Flags().StringVarP(&operation, "operation", "o", "desc", "sort order (asc or desc)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show detailed release information")
	cmd.Flags().IntVarP(&head, "head", "n", 0, "show only the first N releases")
	cmd.Flags().StringVarP(&platform, "platform", "p", currentPlatform, "filter by platform (windows, linux, mac-x64, mac-arm64, server)")
	return cmd
}

func init() {
	versionsCmd.AddCommand(listCmd())
	rootCmd.AddCommand(versionsCmd)
}
