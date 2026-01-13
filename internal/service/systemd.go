package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

const serviceTemplate = `
[Unit]
Description=Belphegor Clipboard Manager
Documentation=https://github.com/labi-le/belphegor

PartOf=graphical-session.target

After=graphical-session.target network.target

Wants=network-online.target

ConditionEnvironment=WAYLAND_DISPLAY

[Service]
Type=simple
ExecStart=%s
Environment="PATH=%s"
Environment="DBUS_SESSION_BUS_ADDRESS=%s"
Restart=on-failure
RestartSec=10

StandardOutput=journal
StandardError=journal

[Install]
WantedBy=graphical-session.target
`

func InstallService(logger zerolog.Logger) error {
	envPath := os.Getenv("PATH")
	if envPath == "" {
		return fmt.Errorf("critical env missing: PATH is empty. Cannot install service")
	}

	envDbus := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if envDbus == "" {
		return fmt.Errorf("critical env missing: DBUS_SESSION_BUS_ADDRESS is empty. Cannot install service")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to detect executable path: %w", err)
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}

	systemdDir := filepath.Join(homeDir, ".config", "systemd", "user")
	serviceFile := filepath.Join(systemdDir, "belphegor.service")

	logger.Info().Msg("try to delete the old service instance")
	_ = runSystemctl(logger, "disable", "--now", "belphegor.service")

	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	quotedPath := absPath
	if strings.Contains(absPath, " ") {
		quotedPath = fmt.Sprintf(`"%s"`, absPath)
	}

	content := fmt.Sprintf(serviceTemplate, quotedPath, envPath, envDbus)

	if err := os.WriteFile(serviceFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	logger.Info().Str("path", serviceFile).Msg("service file created")

	if err := runSystemctl(logger, "daemon-reload"); err != nil {
		return err
	}

	if err := runSystemctl(logger, "enable", "belphegor.service"); err != nil {
		return err
	}

	if err := runSystemctl(logger, "restart", "belphegor.service"); err != nil {
		return err
	}

	logger.Info().Msg("service installed and started successfully")
	return nil
}

func runSystemctl(logger zerolog.Logger, args ...string) error {
	logger.Debug().Strs("args", args).Msg("executing systemctl")

	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl %s failed: %w", strings.Join(args, " "), err)
	}
	return nil
}
