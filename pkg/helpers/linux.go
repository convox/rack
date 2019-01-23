package helpers

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
)

var reLinuxAttributes = regexp.MustCompile(`\s*(\w+)\s*=\s*"?([^"]*)`)

func LinuxRelease() (string, error) {
	attrs, err := linuxReleaseAttributes()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s", attrs["ID"], attrs["VERSION_ID"]), nil
}

func linuxReleaseAttributes() (map[string]string, error) {
	attrs := map[string]string{}

	data, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(bytes.NewReader(data))

	for s.Scan() {
		if p := reLinuxAttributes.FindStringSubmatch(s.Text()); len(p) == 3 {
			attrs[p[1]] = p[2]
		}
	}

	return attrs, nil
}
