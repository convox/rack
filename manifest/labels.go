package manifest

import (
	"fmt"
	"regexp"
	"strconv"
)

var rePortShifters = []*regexp.Regexp{
	regexp.MustCompile(`^convox.port.(\d+).protocol$`),
	regexp.MustCompile(`^convox.port.(\d+).proxy$`),
	regexp.MustCompile(`^convox.port.(\d+).secure$`),
}

// Shift all ports referenced by labels by a given amount
func (labels Labels) Shift(shift int) error {
	for k, v := range labels {
		for _, r := range rePortShifters {
			kn := r.ReplaceAllStringFunc(k, func(s string) string {
				p := r.FindStringSubmatch(k)[1]
				i := r.FindStringSubmatchIndex(k)

				pi, err := strconv.Atoi(p)
				if err != nil {
					return s
				}

				return k[0:i[2]] + fmt.Sprintf("%d", pi+shift) + k[i[3]:]
			})

			if kn != k {
				delete(labels, k)
				labels[kn] = v
			}
		}
	}

	return nil
}
