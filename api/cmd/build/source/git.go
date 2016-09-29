package source

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
)

type SourceGit struct {
	URL string
}

func (s *SourceGit) Fetch(out io.Writer) (string, error) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	u, err := url.Parse(s.URL)
	if err != nil {
		return "", err
	}

	ref := "master"

	if u.Fragment != "" {
		ref = u.Fragment
	}

	repo := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)

	// if we have a user, add it
	if u.User != nil {
		repo = fmt.Sprintf("%s://%s@%s%s", u.Scheme, u.User.String(), u.Host, u.Path)
	}

	switch u.Scheme {
	case "ssh":
		r, err := configureSSH(u)
		if err != nil {
			return "", err
		}

		repo = r
	}

	cmd := exec.Command("git", "clone", "-b", ref, repo, tmp)

	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return tmp, nil
}

func configureSSH(u *url.URL) (string, error) {
	if pw, ok := u.User.Password(); ok {
		key, err := base64.StdEncoding.DecodeString(pw)
		if err != nil {
			return "", err
		}

		err = os.Mkdir("/root/.ssh", 0700)
		if err != nil {
			return "", err
		}

		err = ioutil.WriteFile("/root/.ssh/id_rsa", key, 0400)
		if err != nil {
			return "", err
		}

		os.Setenv("GIT_SSH_COMMAND", "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no")
	}

	return fmt.Sprintf("%s@%s:%s", u.User.Username(), u.Host, u.Path), nil
}
