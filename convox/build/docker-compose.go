package build

func DockerCompose(base string) error {
  err := run("docker-compose", "build")

  if err != nil {
   return err
  }

	return nil
}
