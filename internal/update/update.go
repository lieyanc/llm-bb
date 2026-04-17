package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultOwner = "lieyanc"
	DefaultRepo  = "llm-bb"
	DevTag       = "dev"

	ChannelDev    = "dev"
	ChannelStable = "stable"

	maxArchiveSize = 200 << 20 // 200 MiB
	httpTimeout    = 20 * time.Second
	dlTimeout      = 5 * time.Minute
)

type Release struct {
	TagName         string    `json:"tag_name"`
	Name            string    `json:"name"`
	PublishedAt     time.Time `json:"published_at"`
	Body            string    `json:"body"`
	Prerelease      bool      `json:"prerelease"`
	TargetCommitish string    `json:"target_commitish"`
	Assets          []Asset   `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type CheckResult struct {
	Channel         string    `json:"channel"`
	LatestTag       string    `json:"latestTag"`
	LatestName      string    `json:"latestName"`
	LatestCommit    string    `json:"latestCommit"`
	PublishedAt     time.Time `json:"publishedAt"`
	Notes           string    `json:"notes"`
	AssetName       string    `json:"assetName"`
	AssetSize       int64     `json:"assetSize"`
	UpdateAvailable bool      `json:"updateAvailable"`
}

type Client struct {
	Owner      string
	Repo       string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		Owner:      DefaultOwner,
		Repo:       DefaultRepo,
		HTTPClient: &http.Client{Timeout: httpTimeout},
	}
}

func (c *Client) releaseURL(channel string) (string, error) {
	switch channel {
	case ChannelDev:
		return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", c.Owner, c.Repo, DevTag), nil
	case ChannelStable, "":
		return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", c.Owner, c.Repo), nil
	default:
		return "", fmt.Errorf("unknown channel %q", channel)
	}
}

func (c *Client) FetchRelease(ctx context.Context, channel string) (*Release, error) {
	url, err := c.releaseURL(channel)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no release found for channel %q", channel)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github api %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &release, nil
}

// matchAssets picks the archive + sha256 for the current GOOS/GOARCH.
func matchAssets(assets []Asset) (archive Asset, sha Asset, err error) {
	archiveExt := ".tar.gz"
	if runtime.GOOS == "windows" {
		archiveExt = ".zip"
	}
	suffixArchive := fmt.Sprintf("-%s-%s%s", runtime.GOOS, runtime.GOARCH, archiveExt)
	suffixSHA := fmt.Sprintf("-%s-%s.sha256", runtime.GOOS, runtime.GOARCH)

	var foundArchive, foundSHA bool
	for _, a := range assets {
		switch {
		case strings.HasSuffix(a.Name, suffixArchive):
			archive = a
			foundArchive = true
		case strings.HasSuffix(a.Name, suffixSHA):
			sha = a
			foundSHA = true
		}
	}
	if !foundArchive {
		return archive, sha, fmt.Errorf("no asset for %s/%s (suffix %s)", runtime.GOOS, runtime.GOARCH, suffixArchive)
	}
	if !foundSHA {
		return archive, sha, fmt.Errorf("no sha256 asset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return archive, sha, nil
}

// isUpdateAvailable compares the current build against a release.
func isUpdateAvailable(channel, currentCommit, currentVersion string, release *Release) bool {
	if currentCommit == "" || currentCommit == "unknown" {
		return true
	}
	// Dev channel: compare full commit to release target.
	if channel == ChannelDev {
		if release.TargetCommitish == "" {
			return true
		}
		return !strings.EqualFold(release.TargetCommitish, currentCommit)
	}
	// Stable channel: compare tag strings.
	if currentVersion == "" || currentVersion == "dev" {
		return true
	}
	return !strings.EqualFold(release.TagName, currentVersion)
}

func (c *Client) Check(ctx context.Context, channel, currentCommit, currentVersion string) (*CheckResult, error) {
	release, err := c.FetchRelease(ctx, channel)
	if err != nil {
		return nil, err
	}
	archive, _, err := matchAssets(release.Assets)
	if err != nil {
		return nil, err
	}
	return &CheckResult{
		Channel:         normalizeChannel(channel),
		LatestTag:       release.TagName,
		LatestName:      release.Name,
		LatestCommit:    release.TargetCommitish,
		PublishedAt:     release.PublishedAt,
		Notes:           release.Body,
		AssetName:       archive.Name,
		AssetSize:       archive.Size,
		UpdateAvailable: isUpdateAvailable(channel, currentCommit, currentVersion, release),
	}, nil
}

// Apply downloads the release for the given channel and atomically swaps the running binary.
func (c *Client) Apply(ctx context.Context, channel string) error {
	release, err := c.FetchRelease(ctx, channel)
	if err != nil {
		return err
	}
	archive, shaAsset, err := matchAssets(release.Assets)
	if err != nil {
		return err
	}

	expectedSHA, err := c.downloadSHA(ctx, shaAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("fetch sha256: %w", err)
	}

	archiveData, err := c.download(ctx, archive.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download archive: %w", err)
	}

	actual := sha256.Sum256(archiveData)
	if !strings.EqualFold(hex.EncodeToString(actual[:]), expectedSHA) {
		return fmt.Errorf("sha256 mismatch: want %s, got %s", expectedSHA, hex.EncodeToString(actual[:]))
	}

	binary, err := extractBinary(archive.Name, archiveData)
	if err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}
	if len(binary) == 0 {
		return errors.New("extracted binary is empty")
	}

	return swapBinary(binary)
}

func (c *Client) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: dlTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxArchiveSize))
}

func (c *Client) downloadSHA(ctx context.Context, url string) (string, error) {
	data, err := c.download(ctx, url)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(data))
	if i := strings.IndexAny(line, " \t"); i > 0 {
		line = line[:i]
	}
	if len(line) != 64 {
		return "", fmt.Errorf("invalid sha256 %q", line)
	}
	return strings.ToLower(line), nil
}

func extractBinary(archiveName string, data []byte) ([]byte, error) {
	if strings.HasSuffix(archiveName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(io.LimitReader(rc, maxArchiveSize))
		}
		return nil, errors.New("zip contains no files")
	}
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		return io.ReadAll(io.LimitReader(tr, maxArchiveSize))
	}
	return nil, errors.New("tar contains no files")
}

func swapBinary(newBinary []byte) error {
	current, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(current)
	if err == nil {
		current = resolved
	}

	dir := filepath.Dir(current)
	name := filepath.Base(current)
	newPath := filepath.Join(dir, name+".new")
	oldPath := filepath.Join(dir, name+".old")

	info, err := os.Stat(current)
	if err != nil {
		return fmt.Errorf("stat current: %w", err)
	}

	if err := os.WriteFile(newPath, newBinary, info.Mode()); err != nil {
		return fmt.Errorf("write new binary: %w", err)
	}

	_ = os.Remove(oldPath) // clear any stale leftover

	if err := os.Rename(current, oldPath); err != nil {
		_ = os.Remove(newPath)
		return fmt.Errorf("rename current to .old: %w", err)
	}

	if err := os.Rename(newPath, current); err != nil {
		_ = os.Rename(oldPath, current) // best-effort rollback
		_ = os.Remove(newPath)
		return fmt.Errorf("rename new to current: %w", err)
	}

	// Best-effort. Succeeds on Unix (we can unlink a running exe). Fails on Windows;
	// CleanupOldBinary will remove it on next start.
	_ = os.Remove(oldPath)
	return nil
}

// CleanupOldBinary removes <exe>.old left behind by a previous update. Safe to call at startup.
func CleanupOldBinary() {
	current, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(current); err == nil {
		current = resolved
	}
	_ = os.Remove(filepath.Join(filepath.Dir(current), filepath.Base(current)+".old"))
}

func normalizeChannel(channel string) string {
	if channel == "" {
		return ChannelStable
	}
	return channel
}
