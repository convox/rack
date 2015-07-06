package build

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func query(bin string, args ...string) ([]byte, error) {
	return exec.Command(bin, args...).CombinedOutput()
}

func run(prefix, dir string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err != nil {
		return err
	}

	cmd.Start()

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		fmt.Printf("%s|%s\n", prefix, scanner.Text())
	}

	err = cmd.Wait()

	if err != nil {
		fmt.Printf("%s|error: %s\n", prefix, err)
	}

	return err
}
