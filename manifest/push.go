package manifest

import (
	"fmt"
	"strings"
)

func (m *Manifest) Push(template, app, build string, stream Stream) error {
	for _, s := range m.runOrder() {
		local := fmt.Sprintf("%s/%s", app, s.Name)

		remote := template
		remote = strings.Replace(remote, "{service}", s.Name, -1)
		remote = strings.Replace(remote, "{build}", build, -1)

		if err := DefaultRunner.Run(stream, Docker("tag", local, remote)); err != nil {
			return err
		}

		if err := DefaultRunner.Run(stream, Docker("push", remote)); err != nil {
			return err
		}
	}

	return nil
}
