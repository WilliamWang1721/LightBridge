//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release          *GitHubRelease
	releases         []GitHubRelease
	fetchLatestCalls int
	downloadCalls    int
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(context.Context, string) (*GitHubRelease, error) {
	s.fetchLatestCalls++
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) FetchReleases(context.Context, string, int) ([]GitHubRelease, error) {
	if s.releases != nil {
		return s.releases, nil
	}
	if s.release == nil {
		return nil, nil
	}
	return []GitHubRelease{*s.release}, nil
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	s.downloadCalls++
	return errors.New("DownloadFile should not be called")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	return nil, errors.New("FetchChecksumFile should not be called")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.132",
				Name:    "v0.1.132",
			},
		},
		"0.1.132",
		"release",
	)
	svc.deploymentTypeResolver = func() string { return DeploymentTypeBinary }

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
}

func TestUpdateServiceContainerRejectsBeforeFetchingOrDownloading(t *testing.T) {
	githubClient := &updateServiceGitHubClientStub{
		release: &GitHubRelease{
			TagName: "v0.1.133",
			Name:    "v0.1.133",
		},
	}
	svc := NewUpdateService(&updateServiceCacheStub{}, githubClient, "0.1.132", "release")
	svc.deploymentTypeResolver = func() string { return DeploymentTypeContainer }

	err := svc.PerformUpdateToVersion(context.Background(), "")

	require.ErrorIs(t, err, ErrInPlaceUpdateUnsupported)
	require.Equal(t, 0, githubClient.fetchLatestCalls)
	require.Equal(t, 0, githubClient.downloadCalls)
}

func TestUpdateServiceContainerRollbackRejectsBeforeAccessingExecutable(t *testing.T) {
	svc := NewUpdateService(&updateServiceCacheStub{}, &updateServiceGitHubClientStub{}, "0.1.132", "release")
	svc.deploymentTypeResolver = func() string { return DeploymentTypeContainer }

	err := svc.Rollback()

	require.ErrorIs(t, err, ErrInPlaceUpdateUnsupported)
}

func TestUpdateServiceBinaryDeploymentKeepsExistingUpdatePath(t *testing.T) {
	githubClient := &updateServiceGitHubClientStub{
		release: &GitHubRelease{
			TagName: "v0.1.132",
			Name:    "v0.1.132",
		},
	}
	svc := NewUpdateService(&updateServiceCacheStub{}, githubClient, "0.1.132", "release")
	svc.deploymentTypeResolver = func() string { return DeploymentTypeBinary }

	err := svc.PerformUpdate(context.Background())

	require.ErrorIs(t, err, ErrNoUpdateAvailable)
	require.NotErrorIs(t, err, ErrInPlaceUpdateUnsupported)
	require.Equal(t, 1, githubClient.fetchLatestCalls)
}

func TestDetectUpdateDeploymentTypePrefersExplicitEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		markers map[string]bool
		want    string
	}{
		{
			name:    "explicit container overrides absent marker",
			env:     map[string]string{updateDeploymentTypeEnv: "container"},
			markers: map[string]bool{},
			want:    DeploymentTypeContainer,
		},
		{
			name:    "explicit binary overrides docker marker",
			env:     map[string]string{updateDeploymentTypeEnv: "binary"},
			markers: map[string]bool{"/.dockerenv": true},
			want:    DeploymentTypeBinary,
		},
		{
			name:    "docker marker is detected",
			markers: map[string]bool{"/.dockerenv": true},
			want:    DeploymentTypeContainer,
		},
		{
			name:    "podman marker is detected",
			markers: map[string]bool{"/run/.containerenv": true},
			want:    DeploymentTypeContainer,
		},
		{
			name:    "defaults to binary",
			markers: map[string]bool{},
			want:    DeploymentTypeBinary,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectUpdateDeploymentType(
				func(key string) (string, bool) {
					value, ok := tt.env[key]
					return value, ok
				},
				func(path string) bool { return tt.markers[path] },
			)

			require.Equal(t, tt.want, got)
		})
	}
}

func TestUpdateInfoSerializesDeploymentCapabilities(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{release: &GitHubRelease{TagName: "v0.1.133", Name: "v0.1.133"}},
		"0.1.132",
		"release",
	)
	svc.deploymentTypeResolver = func() string { return DeploymentTypeContainer }

	info, err := svc.CheckUpdate(context.Background(), true)
	require.NoError(t, err)

	payload, err := json.Marshal(info)
	require.NoError(t, err)
	var decoded struct {
		Capabilities UpdateCapabilities `json:"capabilities"`
	}
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, DeploymentTypeContainer, decoded.Capabilities.DeploymentType)
	require.False(t, decoded.Capabilities.CanInPlaceUpdate)
	require.False(t, decoded.Capabilities.CanRollback)
	require.False(t, decoded.Capabilities.CanRestart)
}

func TestUpdateServiceListVersionReleasesIncludesHistory(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.1",
				Name:    "v0.1.1",
				Body:    "latest",
			},
			releases: []GitHubRelease{
				{
					TagName:     "v0.1.1",
					Name:        "v0.1.1",
					Body:        "latest",
					PublishedAt: "2026-06-07T00:00:00Z",
					HTMLURL:     "https://example.test/releases/v0.1.1",
				},
				{
					TagName:     "v0.0.1",
					Name:        "v0.0.1",
					Body:        "first",
					PublishedAt: "2026-05-01T00:00:00Z",
					HTMLURL:     "https://example.test/releases/v0.0.1",
				},
			},
		},
		"0.0.1",
		"release",
	)

	releases, info, err := svc.ListVersionReleases(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, "0.1.1", info.LatestVersion)
	require.Len(t, releases, 2)
	require.Equal(t, "0.1.1", releases[0].Version)
	require.True(t, releases[0].Latest)
	require.False(t, releases[0].Current)
	require.Equal(t, "0.0.1", releases[1].Version)
	require.True(t, releases[1].Current)
	require.False(t, releases[1].Latest)
}

func TestUpdateServiceListVersionReleasesIncludesPreview(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{TagName: "v0.2.4", Name: "v0.2.4"},
			releases: []GitHubRelease{
				{
					TagName:     "v0.2.4-preview",
					Name:        "v0.2.4-preview",
					Prerelease:  true,
					PublishedAt: "2026-06-10T00:00:00Z",
				},
				{
					TagName:     "v0.2.4",
					Name:        "v0.2.4",
					PublishedAt: "2026-06-09T00:00:00Z",
				},
			},
		},
		"0.2.3",
		"release",
	)

	releases, _, err := svc.ListVersionReleases(context.Background(), true)

	require.NoError(t, err)
	require.Len(t, releases, 2)
	require.Equal(t, "0.2.4-preview", releases[0].Version)
	require.True(t, releases[0].Prerelease)
	require.Equal(t, "0.2.4", releases[1].Version)
	require.False(t, releases[1].Prerelease)
}

func TestUpdateServicePreviewProductionSameNumberAreInstallable(t *testing.T) {
	// 当前为正式版 0.2.4，应允许安装同号的 preview 0.2.4-preview（HasUpdate=true）。
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			releases: []GitHubRelease{
				{TagName: "v0.2.4-preview", Name: "v0.2.4-preview", Prerelease: true},
				{TagName: "v0.2.4", Name: "v0.2.4"},
			},
		},
		"0.2.4",
		"release",
	)

	info, err := svc.updateInfoForTargetVersion(context.Background(), "0.2.4-preview")

	require.NoError(t, err)
	require.True(t, info.HasUpdate)
	require.Equal(t, "0.2.4-preview", info.LatestVersion)
}

func TestUpdateServiceDetectsPreviewPatchUpdate(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName:    "v0.2.9-preview.1",
				Name:       "v0.2.9-preview.1",
				Prerelease: true,
			},
		},
		"0.2.9-preview",
		"release",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.True(t, info.HasUpdate)
	require.Equal(t, "0.2.9-preview.1", info.LatestVersion)
}

func TestParseVersionStripsPrereleaseSuffix(t *testing.T) {
	tests := []struct {
		version string
		want    [3]int
	}{
		{version: "0.2.4-preview", want: [3]int{0, 2, 4}},
		{version: "v0.2.4-rc.1", want: [3]int{0, 2, 4}},
		{version: "1.2.3+build.5", want: [3]int{1, 2, 3}},
	}
	for _, tt := range tests {
		parts, _ := parseSemanticVersion(tt.version)
		require.Equal(t, tt.want, parts)
	}
}

func TestCompareVersionsHandlesPrereleaseIdentifiers(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
	}{
		{name: "preview patch is newer", current: "0.2.9-preview", latest: "0.2.9-preview.1", want: -1},
		{name: "next patch preview is newer", current: "0.2.9-preview.1", latest: "0.2.10-preview", want: -1},
		{name: "release is newer than preview", current: "0.2.4-preview", latest: "0.2.4", want: -1},
		{name: "preview is older than release", current: "0.2.4", latest: "0.2.4-preview", want: 1},
		{name: "numeric prerelease segments sort numerically", current: "0.2.4-rc.10", latest: "0.2.4-rc.2", want: 1},
		{name: "build metadata is ignored", current: "1.2.3+build.1", latest: "1.2.3+build.2", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, compareVersions(tt.current, tt.latest))
		})
	}
}

func TestSelectUpdateAssetPreviewPrefersIncrementalBinary(t *testing.T) {
	svc := NewUpdateService(&updateServiceCacheStub{}, &updateServiceGitHubClientStub{}, "0.2.3", "release")
	archiveName := fmt.Sprintf("LightBridge_0.2.4-preview_%s.tar.gz", svc.getArchiveName())
	binaryName := fmt.Sprintf("LightBridge_0.2.4-preview_%s", svc.getArchiveName())
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	info := &UpdateInfo{
		LatestVersion: "0.2.4-preview",
		ReleaseInfo: &ReleaseInfo{
			Prerelease: true,
			Assets: []Asset{
				{Name: archiveName, DownloadURL: "https://github.com/example/archive"},
				{Name: binaryName, DownloadURL: "https://github.com/example/binary"},
				{Name: "checksums.txt", DownloadURL: "https://github.com/example/checksums"},
			},
		},
	}

	asset, checksumURL, directBinary := svc.selectUpdateAsset(info)

	require.NotNil(t, asset)
	require.Equal(t, binaryName, asset.Name)
	require.Equal(t, "https://github.com/example/checksums", checksumURL)
	require.True(t, directBinary)
}

func TestSelectUpdateAssetProductionUsesArchive(t *testing.T) {
	svc := NewUpdateService(&updateServiceCacheStub{}, &updateServiceGitHubClientStub{}, "0.2.3", "release")
	binaryName := fmt.Sprintf("LightBridge_0.2.4_%s", svc.getArchiveName())
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	archiveName := fmt.Sprintf("LightBridge_0.2.4_%s.tar.gz", svc.getArchiveName())
	info := &UpdateInfo{
		LatestVersion: "0.2.4",
		ReleaseInfo: &ReleaseInfo{
			Prerelease: false,
			Assets: []Asset{
				{Name: binaryName, DownloadURL: "https://github.com/example/binary"},
				{Name: archiveName, DownloadURL: "https://github.com/example/archive"},
			},
		},
	}

	asset, _, directBinary := svc.selectUpdateAsset(info)

	require.NotNil(t, asset)
	require.Equal(t, archiveName, asset.Name)
	require.False(t, directBinary)
}

func TestSelectUpdateAssetPreviewFallsBackToArchive(t *testing.T) {
	svc := NewUpdateService(&updateServiceCacheStub{}, &updateServiceGitHubClientStub{}, "0.2.3", "release")
	archiveName := fmt.Sprintf("LightBridge_0.2.4-preview_%s.tar.gz", svc.getArchiveName())
	info := &UpdateInfo{
		LatestVersion: "0.2.4-preview",
		ReleaseInfo: &ReleaseInfo{
			Prerelease: true,
			Assets: []Asset{
				{Name: archiveName, DownloadURL: "https://github.com/example/archive"},
			},
		},
	}

	asset, _, directBinary := svc.selectUpdateAsset(info)

	require.NotNil(t, asset)
	require.Equal(t, archiveName, asset.Name)
	require.False(t, directBinary)
}
