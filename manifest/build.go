package manifest

func (m *Manifest) Build(dir string) error {
	builds := map[string]string{}
	pulls := map[string]string{}

	for _, service := range m.Services {
		switch {
		case service.Build != "":
			builds[service.Build] = service.Tag()
		case service.Image != "":
			pulls[service.Image] = service.Tag()
		}
	}

	for dir, tag := range builds {
		args := []string{"build"}

		args = append(args, "-t", tag)
		args = append(args, dir)

		runPrefix(systemPrefix(m), Docker(args...))
	}

	for image, tag := range pulls {
		args := []string{"pull"}

		args = append(args, image)

		runPrefix(systemPrefix(m), Docker(args...))
		runPrefix(systemPrefix(m), Docker("tag", image, tag))
	}

	return nil
}
