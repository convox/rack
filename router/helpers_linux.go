package router

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
)

var kvpair = regexp.MustCompile(`(.*)=([^"].*|\"(.*)\")`)

func linuxReleaseAttributes() (map[string]string, error) {
	attrs := map[string]string{}
	data, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		p := kvpair.FindStringSubmatch(s.Text())
		if len(p) == 4 {
			if p[3] != "" {
				attrs[p[1]] = p[3]
			} else {
				attrs[p[1]] = p[2]
			}
		}
	}
	return attrs, nil
}

func linuxRelease() (string, error) {
	attrs, err := linuxReleaseAttributes()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", attrs["ID"], attrs["VERSION_ID"]), nil
}
