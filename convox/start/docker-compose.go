package start

func DockerCompose(base string) error {
	err := run("docker-compose", "up")

	if err != nil {
		return err
	}

	return nil
}
