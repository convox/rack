package source

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os/exec"
	"time"
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

	cmd := exec.Command("git", "clone", "-b", ref, fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path), tmp)
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	time.Sleep(10 * time.Second)

	return tmp, nil
}
