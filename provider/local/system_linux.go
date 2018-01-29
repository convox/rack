package local

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func launcherPath(name string) string {
	return filepath.Join("/lib/systemd/system", fmt.Sprintf("%s.service", name))
}

func launcherStart(name string) error {
	return exec.Command("systemctl", "start", name).Run()
}

func launcherStop(name string) error {
	return exec.Command("systemctl", "stop", name).Run()
}

func launcherTemplate() string {
	return `
[Unit]
After=network.target

[Service]
Type=simple
ExecStart={{ .Command }} {{ range .Args }}{{ . }} {{ end }}
KillMode=control-group
Restart=always
RestartSec=10s

[Install]
WantedBy=default.target`
}
