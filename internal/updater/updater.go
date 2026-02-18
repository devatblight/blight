package updater

import (
	"fmt"
	"os"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

type Updater struct {
	repo string
}

func New(repo string) *Updater {
	return &Updater{repo: repo}
}

func (u *Updater) CheckForUpdates(currentVersion string) (*selfupdate.Release, bool, error) {
	v, err := semver.Parse(currentVersion)
	if err != nil {
		return nil, false, fmt.Errorf("failed to parse current version: %w", err)
	}

	latest, found, err := selfupdate.DetectLatest(u.repo)
	if err != nil {
		return nil, false, fmt.Errorf("check for updates failed: %w", err)
	}
	if !found {
		return nil, false, nil // No release found
	}

	if latest.Version.GT(v) {
		return latest, true, nil
	}
	return nil, false, nil
}

// ApplyUpdate downloads and applies the update from the given release.
func (u *Updater) ApplyUpdate(release *selfupdate.Release) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate executable path: %w", err)
	}

	if err := selfupdate.UpdateTo(release.AssetURL, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}
	return nil
}
