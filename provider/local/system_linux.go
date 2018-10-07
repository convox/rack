package local

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func launcherPath(name string) string {
	return filepath.Join("/lib/systemd/system", fmt.Sprintf("convox.%s.service", name))
}

func launcherStart(name string) error {
	return exec.Command("systemctl", "start", fmt.Sprintf("convox.%s", name)).Run()
}

func launcherStop(name string) error {
	return exec.Command("systemctl", "stop", fmt.Sprintf("convox.%s", name)).Run()
}

func launcherTemplate() string {
	return `
[Unit]
After=network.target

[Service]
Type=simple
Environment='PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin"'
ExecStart={{ .Command }} {{ range .Args }}{{ . }} {{ end }}
KillMode=control-group
Restart=always
RestartSec=10s

[Install]
WantedBy=default.target`
}
