package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckStableUsesSemverAndBareBinaryAsset(t *testing.T) {
	binaryName := targetName()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Release{
			TagName: "v1.2.0",
			Name:    "Release v1.2.0",
			Assets: []Asset{
				{Name: binaryName, BrowserDownloadURL: serverURL(r, "/download/"+binaryName), Size: 12},
				{Name: binaryName + ".sha256", BrowserDownloadURL: serverURL(r, "/download/"+binaryName+".sha256")},
			},
		})
	}))
	defer server.Close()
	withGitHubAPIBaseURL(t, server.URL)

	client := testClient(t, server)
	result, err := client.Check(context.Background(), ChannelStable, "aaaaaaa", "v1.1.0")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected v1.2.0 to update v1.1.0")
	}
	if result.AssetName != binaryName {
		t.Fatalf("expected bare binary asset %q, got %q", binaryName, result.AssetName)
	}

	result, err = client.Check(context.Background(), ChannelStable, "aaaaaaa", "v1.3.0")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.UpdateAvailable {
		t.Fatalf("did not expect older stable release to be available")
	}
}

func TestCheckDevLoadsVersionMetadata(t *testing.T) {
	binaryName := targetName()
	remoteVersion := "dev-0042-20260608-bbbbbbb"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/owner/repo/releases/download/dev/version.json":
			_ = json.NewEncoder(w).Encode(versionMetadata{
				Version:   remoteVersion,
				Commit:    "bbbbbbb",
				BuildDate: "2026-06-08T00:00:00Z",
				Channel:   "dev",
				Tag:       "dev",
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	withGitHubBaseURL(t, server.URL)

	client := testClient(t, server)
	result, err := client.Check(context.Background(), ChannelDev, "aaaaaaa", "dev-0041-20260607-aaaaaaa")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected dev metadata commit to mark update available")
	}
	if result.LatestVersion != remoteVersion {
		t.Fatalf("expected latest version %q, got %q", remoteVersion, result.LatestVersion)
	}
	if result.LatestCommit != "bbbbbbb" {
		t.Fatalf("expected metadata commit, got %q", result.LatestCommit)
	}
	if result.LatestBuildDate != "2026-06-08T00:00:00Z" {
		t.Fatalf("expected metadata build date, got %q", result.LatestBuildDate)
	}
	if result.AssetName != binaryName {
		t.Fatalf("expected dev OTA asset %q, got %q", binaryName, result.AssetName)
	}
}

func TestMatchAssetsFallsBackToArchive(t *testing.T) {
	archiveName := fmt.Sprintf("llm-bb-v1.0.0-%s-%s.tar.gz", runtimeGOOS(), runtimeGOARCH())
	if runtimeGOOS() == "windows" {
		archiveName = fmt.Sprintf("llm-bb-v1.0.0-%s-%s.zip", runtimeGOOS(), runtimeGOARCH())
	}
	assets, err := matchAssets([]Asset{
		{Name: archiveName},
		{Name: archiveName + ".sha256"},
	})
	if err != nil {
		t.Fatalf("matchAssets returned error: %v", err)
	}
	if assets.kind != assetKindArchive || !assets.needsUnzip {
		t.Fatalf("expected archive fallback, got kind=%q needsUnzip=%v", assets.kind, assets.needsUnzip)
	}
	if assets.sha.Name != archiveName+".sha256" {
		t.Fatalf("expected exact archive checksum, got %q", assets.sha.Name)
	}
}

func TestDownloadSHAParsesSha256sumOutput(t *testing.T) {
	want := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(want + "  llm-bb-linux-amd64\n"))
	}))
	defer server.Close()

	got, err := testClient(t, server).downloadSHA(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("downloadSHA returned error: %v", err)
	}
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	const content = "new binary"
	data := tarGz(t, "llm-bb", []byte(content))
	got, err := extractBinary("llm-bb-linux-amd64.tar.gz", data)
	if err != nil {
		t.Fatalf("extractBinary returned error: %v", err)
	}
	if string(got) != content {
		t.Fatalf("expected %q, got %q", content, got)
	}
}

func TestIsUpdateAvailableDevComparisons(t *testing.T) {
	release := &Release{
		TagName: "dev",
		Version: "dev-0042-20260608-bbbbbbb",
		Commit:  "bbbbbbb",
	}
	if !isUpdateAvailable(ChannelDev, "aaaaaaa", "dev-0041-20260607-aaaaaaa", release) {
		t.Fatalf("expected different dev commit to be available")
	}
	if isUpdateAvailable(ChannelDev, "bbbbbbb", "dev-0042-20260608-bbbbbbb", release) {
		t.Fatalf("did not expect identical dev commit to be available")
	}
	if isUpdateAvailable(ChannelDev, "unknown", "dev-0043-20260609-ccccccc", release) {
		t.Fatalf("did not expect lower run number to update newer dev build")
	}
}

func testClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	oldAPIBaseURL := githubAPIBaseURL
	oldBaseURL := githubBaseURL
	githubAPIBaseURL = server.URL
	githubBaseURL = server.URL
	t.Cleanup(func() {
		githubAPIBaseURL = oldAPIBaseURL
		githubBaseURL = oldBaseURL
	})

	return &Client{
		Owner:      "owner",
		Repo:       "repo",
		HTTPClient: server.Client(),
	}
}

func withGitHubBaseURL(t *testing.T, baseURL string) {
	t.Helper()
	original := githubBaseURL
	githubBaseURL = baseURL
	t.Cleanup(func() {
		githubBaseURL = original
	})
}

func withGitHubAPIBaseURL(t *testing.T, baseURL string) {
	t.Helper()
	original := githubAPIBaseURL
	githubAPIBaseURL = baseURL
	t.Cleanup(func() {
		githubAPIBaseURL = original
	})
}

func serverURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}

func tarGz(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	if err := tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(body)),
	}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("write tar body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

func runtimeGOOS() string {
	return strings.Split(targetName(), "-")[2]
}

func runtimeGOARCH() string {
	name := strings.TrimSuffix(targetName(), ".exe")
	parts := strings.Split(name, "-")
	return parts[len(parts)-1]
}
