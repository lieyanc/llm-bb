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
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

var (
	githubAPIBaseURL = "https://api.github.com"
	githubBaseURL    = "https://github.com"
)

type Release struct {
	TagName         string    `json:"tag_name"`
	Name            string    `json:"name"`
	PublishedAt     time.Time `json:"published_at"`
	Body            string    `json:"body"`
	Prerelease      bool      `json:"prerelease"`
	TargetCommitish string    `json:"target_commitish"`
	Assets          []Asset   `json:"assets"`
	Version         string
	Commit          string
	BuildDate       string
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type CheckResult struct {
	Channel         string     `json:"channel"`
	LatestTag       string     `json:"latestTag"`
	LatestName      string     `json:"latestName"`
	LatestVersion   string     `json:"latestVersion"`
	LatestCommit    string     `json:"latestCommit"`
	LatestBuildDate string     `json:"latestBuildDate"`
	PublishedAt     *time.Time `json:"publishedAt,omitempty"`
	Notes           string     `json:"notes"`
	AssetName       string     `json:"assetName"`
	AssetSize       int64      `json:"assetSize"`
	UpdateAvailable bool       `json:"updateAvailable"`
}

type Client struct {
	Owner            string
	Repo             string
	HTTPClient       *http.Client
	APITimeout       time.Duration
	DownloadTimeout  time.Duration
	MaxDownloadBytes int64
}

type versionMetadata struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	BuildTime string `json:"build_time"`
	Channel   string `json:"channel"`
	Tag       string `json:"tag"`
}

type matchedAssets struct {
	payload    Asset
	sha        Asset
	kind       assetKind
	needsUnzip bool
}

type assetKind string

const (
	assetKindBinary  assetKind = "binary"
	assetKindArchive assetKind = "archive"
)

func NewClient() *Client {
	return &Client{
		Owner:            DefaultOwner,
		Repo:             DefaultRepo,
		APITimeout:       httpTimeout,
		DownloadTimeout:  dlTimeout,
		MaxDownloadBytes: maxArchiveSize,
	}
}

func (c *Client) httpClient(timeout time.Duration) *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	if timeout <= 0 {
		timeout = httpTimeout
	}
	return &http.Client{Timeout: timeout}
}

func (c *Client) apiTimeout() time.Duration {
	if c.APITimeout <= 0 {
		return httpTimeout
	}
	return c.APITimeout
}

func (c *Client) downloadTimeout() time.Duration {
	if c.DownloadTimeout <= 0 {
		return dlTimeout
	}
	return c.DownloadTimeout
}

func (c *Client) maxDownloadBytes() int64 {
	if c.MaxDownloadBytes <= 0 {
		return maxArchiveSize
	}
	return c.MaxDownloadBytes
}

func (c *Client) releaseURL(channel string) (string, error) {
	switch channel {
	case ChannelDev:
		return fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", strings.TrimRight(githubAPIBaseURL, "/"), c.Owner, c.Repo, DevTag), nil
	case ChannelStable, "":
		return fmt.Sprintf("%s/repos/%s/%s/releases/latest", strings.TrimRight(githubAPIBaseURL, "/"), c.Owner, c.Repo), nil
	default:
		return "", fmt.Errorf("unknown channel %q", channel)
	}
}

func (c *Client) FetchRelease(ctx context.Context, channel string) (*Release, error) {
	if normalizeChannel(channel) == ChannelDev {
		return c.fetchDevRelease(ctx)
	}

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

	resp, err := c.httpClient(c.apiTimeout()).Do(req)
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

func (c *Client) fetchDevRelease(ctx context.Context) (*Release, error) {
	binaryName := targetName()
	release := &Release{
		TagName:    DevTag,
		Name:       "Development Build",
		Prerelease: true,
		Assets: []Asset{
			{
				Name:               binaryName,
				BrowserDownloadURL: c.releaseAssetURL(DevTag, binaryName),
			},
			{
				Name:               binaryName + ".sha256",
				BrowserDownloadURL: c.releaseAssetURL(DevTag, binaryName+".sha256"),
			},
			{
				Name:               "version.json",
				BrowserDownloadURL: c.releaseAssetURL(DevTag, "version.json"),
			},
		},
	}
	if err := c.loadVersionMetadata(ctx, release); err != nil {
		return nil, err
	}
	return release, nil
}

func (c *Client) releaseAssetURL(tag, assetName string) string {
	return strings.TrimRight(githubBaseURL, "/") +
		"/" + url.PathEscape(c.Owner) +
		"/" + url.PathEscape(c.Repo) +
		"/releases/download/" + url.PathEscape(tag) +
		"/" + url.PathEscape(assetName)
}

func (c *Client) loadVersionMetadata(ctx context.Context, release *Release) error {
	versionAsset, ok := findAssetByName(release.Assets, "version.json")
	if !ok {
		return fmt.Errorf("version.json asset not found in %s release", release.TagName)
	}
	data, err := c.download(ctx, versionAsset.BrowserDownloadURL, 16*1024)
	if err != nil {
		return fmt.Errorf("download version metadata: %w", err)
	}
	var meta versionMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("decode version metadata: %w", err)
	}
	if meta.Version = strings.TrimSpace(meta.Version); meta.Version != "" {
		release.Version = meta.Version
	}
	if meta.Commit = strings.TrimSpace(meta.Commit); meta.Commit != "" {
		release.Commit = meta.Commit
	}
	buildDate := strings.TrimSpace(meta.BuildDate)
	if buildDate == "" {
		buildDate = strings.TrimSpace(meta.BuildTime)
	}
	if buildDate != "" {
		release.BuildDate = buildDate
	}
	if tag := strings.TrimSpace(meta.Tag); tag != "" {
		release.TagName = tag
	}
	if release.TargetCommitish == "" && release.Commit != "" {
		release.TargetCommitish = release.Commit
	}
	return nil
}

func findAssetByName(assets []Asset, name string) (Asset, bool) {
	for _, a := range assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

func targetName() string {
	name := fmt.Sprintf("llm-bb-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// matchAssets picks the OTA binary + sha256 for the current GOOS/GOARCH.
// It falls back to the old archive format so existing releases stay usable.
func matchAssets(assets []Asset) (matchedAssets, error) {
	binaryName := targetName()
	if binary, ok := findAssetByName(assets, binaryName); ok {
		sha, ok := findAssetByName(assets, binaryName+".sha256")
		if !ok {
			return matchedAssets{}, fmt.Errorf("no sha256 asset for %s", binaryName)
		}
		return matchedAssets{
			payload: binary,
			sha:     sha,
			kind:    assetKindBinary,
		}, nil
	}

	archiveExt := ".tar.gz"
	if runtime.GOOS == "windows" {
		archiveExt = ".zip"
	}
	suffixArchive := fmt.Sprintf("-%s-%s%s", runtime.GOOS, runtime.GOARCH, archiveExt)
	suffixSHA := fmt.Sprintf("-%s-%s.sha256", runtime.GOOS, runtime.GOARCH)

	var archive Asset
	var sha Asset
	var foundArchive bool
	for _, a := range assets {
		if strings.HasSuffix(a.Name, suffixArchive) {
			archive = a
			foundArchive = true
		}
	}
	if !foundArchive {
		return matchedAssets{}, fmt.Errorf("no asset for %s/%s (wanted %s or suffix %s)", runtime.GOOS, runtime.GOARCH, binaryName, suffixArchive)
	}

	var foundSHA bool
	if exactSHA, ok := findAssetByName(assets, archive.Name+".sha256"); ok {
		sha = exactSHA
		foundSHA = true
	} else {
		for _, a := range assets {
			if strings.HasSuffix(a.Name, suffixSHA) {
				sha = a
				foundSHA = true
				break
			}
		}
	}
	if !foundSHA {
		return matchedAssets{}, fmt.Errorf("no sha256 asset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return matchedAssets{
		payload:    archive,
		sha:        sha,
		kind:       assetKindArchive,
		needsUnzip: true,
	}, nil
}

// isUpdateAvailable compares the current build against a release.
func isUpdateAvailable(channel, currentCommit, currentVersion string, release *Release) bool {
	if currentVersion == "" || currentVersion == "dev" {
		return true
	}
	if normalizeChannel(channel) == ChannelDev {
		remoteCommit := normalizeCommit(release.Commit)
		if remoteCommit == "" {
			remoteCommit = normalizeCommit(release.TargetCommitish)
		}
		currentCommit = normalizeCommit(currentCommit)
		if remoteCommit != "" && currentCommit != "" {
			return remoteCommit != currentCommit
		}
		remoteVersion := release.DisplayVersion()
		remoteRun, remoteSHA := parseDevVersion(remoteVersion)
		localRun, localSHA := parseDevVersion(currentVersion)
		if remoteSHA != "" && localSHA != "" && remoteSHA == localSHA {
			return false
		}
		if remoteRun > 0 && localRun > 0 {
			return remoteRun > localRun
		}
		return false
	}
	return semverGreater(release.TagName, currentVersion)
}

func (r Release) DisplayVersion() string {
	if strings.TrimSpace(r.Version) != "" {
		return strings.TrimSpace(r.Version)
	}
	return r.TagName
}

func normalizeCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if commit == "" || commit == "unknown" {
		return ""
	}
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

func semverGreater(a, b string) bool {
	av := parseSemver(strings.TrimPrefix(a, "v"))
	bv := parseSemver(strings.TrimPrefix(b, "v"))
	for i := 0; i < 3; i++ {
		if av[i] > bv[i] {
			return true
		}
		if av[i] < bv[i] {
			return false
		}
	}
	return false
}

func parseSemver(s string) [3]int {
	var result [3]int
	parts := strings.SplitN(s, ".", 3)
	for i, p := range parts {
		if i >= 3 {
			break
		}
		if idx := strings.IndexByte(p, '-'); idx >= 0 {
			p = p[:idx]
		}
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}

func parseDevVersion(v string) (runNumber int, sha string) {
	parts := strings.SplitN(v, "-", 4)
	if len(parts) >= 4 && parts[0] == "dev" {
		n, _ := strconv.Atoi(parts[1])
		return n, parts[3]
	}
	return 0, ""
}

func (c *Client) Check(ctx context.Context, channel, currentCommit, currentVersion string) (*CheckResult, error) {
	release, err := c.FetchRelease(ctx, channel)
	if err != nil {
		return nil, err
	}
	assets, err := matchAssets(release.Assets)
	if err != nil {
		return nil, err
	}
	latestCommit := release.Commit
	if latestCommit == "" {
		latestCommit = release.TargetCommitish
	}
	var publishedAt *time.Time
	if !release.PublishedAt.IsZero() {
		publishedAt = &release.PublishedAt
	}
	return &CheckResult{
		Channel:         normalizeChannel(channel),
		LatestTag:       release.TagName,
		LatestName:      release.Name,
		LatestVersion:   release.DisplayVersion(),
		LatestCommit:    latestCommit,
		LatestBuildDate: release.BuildDate,
		PublishedAt:     publishedAt,
		Notes:           release.Body,
		AssetName:       assets.payload.Name,
		AssetSize:       assets.payload.Size,
		UpdateAvailable: isUpdateAvailable(channel, currentCommit, currentVersion, release),
	}, nil
}

// Apply downloads the release for the given channel and atomically swaps the running binary.
func (c *Client) Apply(ctx context.Context, channel string) error {
	release, err := c.FetchRelease(ctx, channel)
	if err != nil {
		return err
	}
	assets, err := matchAssets(release.Assets)
	if err != nil {
		return err
	}

	expectedSHA, err := c.downloadSHA(ctx, assets.sha.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("fetch sha256: %w", err)
	}

	payload, err := c.download(ctx, assets.payload.BrowserDownloadURL, c.maxDownloadBytes())
	if err != nil {
		return fmt.Errorf("download %s: %w", assets.kind, err)
	}

	actual := sha256.Sum256(payload)
	if !strings.EqualFold(hex.EncodeToString(actual[:]), expectedSHA) {
		return fmt.Errorf("sha256 mismatch: want %s, got %s", expectedSHA, hex.EncodeToString(actual[:]))
	}

	binary := payload
	if assets.needsUnzip {
		binary, err = extractBinary(assets.payload.Name, payload, c.maxDownloadBytes())
		if err != nil {
			return fmt.Errorf("extract binary: %w", err)
		}
	}
	if len(binary) == 0 {
		return errors.New("extracted binary is empty")
	}

	return swapBinary(binary)
}

func (c *Client) download(ctx context.Context, url string, maxSize int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient(c.downloadTimeout()).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxSize))
}

func (c *Client) downloadSHA(ctx context.Context, url string) (string, error) {
	data, err := c.download(ctx, url, 1024)
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

func extractBinary(archiveName string, data []byte, maxSize int64) ([]byte, error) {
	if maxSize <= 0 {
		maxSize = maxArchiveSize
	}
	if strings.HasSuffix(archiveName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		var firstRegular []byte
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			content, readErr := io.ReadAll(io.LimitReader(rc, maxSize))
			closeErr := rc.Close()
			if readErr != nil {
				return nil, readErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			if isPreferredBinary(f.Name) {
				return content, nil
			}
			if firstRegular == nil {
				firstRegular = content
			}
		}
		if firstRegular != nil {
			return firstRegular, nil
		}
		return nil, errors.New("zip contains no files")
	}
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	var firstRegular []byte
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
		content, err := io.ReadAll(io.LimitReader(tr, maxSize))
		if err != nil {
			return nil, err
		}
		if isPreferredBinary(hdr.Name) {
			return content, nil
		}
		if firstRegular == nil {
			firstRegular = content
		}
	}
	if firstRegular != nil {
		return firstRegular, nil
	}
	return nil, errors.New("tar contains no files")
}

func isPreferredBinary(name string) bool {
	want := "llm-bb"
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	return filepath.Base(name) == want
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
