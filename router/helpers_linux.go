package router

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

var kvpair = regexp.MustCompile(`(.*[^=])=(.*)`)

func linuxReleaseAttributes() (map[string]string, error) {
	attrs := map[string]string{}
	data, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		line := s.Text()
		line = strings.Replace(line, "\"", "", -1)
		parts := kvpair.FindAllStringSubmatch(line, -1)
		for _, kv := range parts {
			attrs[kv[1]] = kv[2]
		}
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
