package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const AppName = "Blight"

// IsInstalled checks if the current executable is in the Local AppData directory.
func IsInstalled() (bool, error) {
	exePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get executable path: %w", err)
	}

	installDir, err := GetInstallDir()
	if err != nil {
		return false, err
	}

	// Normalize paths for comparison
	absExe, err := filepath.Abs(exePath)
	if err != nil {
		return false, err
	}
	absInstall, err := filepath.Abs(installDir)
	if err != nil {
		return false, err
	}

	// Check if the executable is within the install directory
	return strings.HasPrefix(strings.ToLower(absExe), strings.ToLower(absInstall)), nil
}

// GetInstallDir returns the installation directory in LocalAppData.
func GetInstallDir() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		// Fallback to UserConfigDir (Roaming) if LocalAppData is not set
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user config dir: %w", err)
		}
		localAppData = dir
	}

	return filepath.Join(localAppData, AppName), nil
}

// Install copies the current executable to the installation directory.
// It returns the path to the installed executable.
func Install() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	installDir, err := GetInstallDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create install dir: %w", err)
	}

	destPath := filepath.Join(installDir, filepath.Base(exePath))

	// If source and dest are the same, we are already installed
	if strings.EqualFold(exePath, destPath) {
		return destPath, nil
	}

	// Copy file
	srcFile, err := os.Open(exePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return destPath, nil
}
