package router

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
)

func linuxReleaseAttributes() (map[string]string, error) {
	attrs := map[string]string{}
	data, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		line := s.Text()
		if len(line) == 0 {
			continue
		}
		line = strings.Replace(line, "\"", "", -1)
		line = strings.ToLower(line)
		parts := strings.Split(line, "=")
		attrs[parts[0]] = parts[1]
	}
	return attrs, nil
}

func linuxRelease() (string, error) {
	attrs, err := linuxReleaseAttributes()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", attrs["NAME"], attrs["VERSION_ID"]), nil
}
