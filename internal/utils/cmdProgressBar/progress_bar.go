package cmdProgressBar

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"

	"github.com/Tar-Mairon24/vslauncher/internal/download"
)

var theme = progressbar.Theme{
	Saucer:        color.New(color.FgMagenta).Sprint("="),
	SaucerHead:    color.New(color.FgMagenta, color.Bold).Sprint(">"),
	SaucerPadding: " ",
	BarStart:      "[",
	BarEnd:        "]",
}

const (
	throttleRate = 65 * time.Millisecond
)

func newDownloadBar(size int64, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		size,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(color.Output),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetTheme(theme),
		progressbar.OptionShowBytes(true),
		progressbar.OptionThrottle(throttleRate),
		progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
	)
}

func newExtractBar(total int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(
		total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(color.Output),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetTheme(theme),
		progressbar.OptionShowCount(),
		progressbar.OptionThrottle(throttleRate),
		progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
	)
}

type barStepReporter struct {
	bar *progressbar.ProgressBar
}

func (r *barStepReporter) Step(label string) {
	r.bar.Describe("extracting " + filepath.Base(label))
	r.bar.Add(1)
}

func New(downloadLabel string) download.ProgressReporters {
	return download.ProgressReporters{
		Download: func(size int64) io.Writer {
			return newDownloadBar(size, downloadLabel)
		},
		Extract: func(total int) download.StepReporter {
			bar := newExtractBar(total, "extracting files")
			return &barStepReporter{bar: bar}
		},
	}
}
