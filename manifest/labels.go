package manifest

import (
	"fmt"
	"strconv"
	"strings"
)


// Trim "convox.port." and ".protocol" from beginning and end of label
//
func extractPort(label string) string {
	label = strings.TrimSuffix(label, ".protocol")
	label = strings.TrimPrefix(label, "convox.port.")
	return label
}

// "convox.port.xxx.protocol" label can be set to https.
// Increment the 'xxx' part by the given shift amount.
//
func (labels Labels) Shift(shift int) Labels {

	// Make a copy of the old labels, since we can't update them on the fly
	oldlabels := make(map[string]string)
	for k,v := range labels {
	  oldlabels[k] = v
	}

	for ol := range oldlabels {

		// If we have a label called 'convox.port.xxx.protocol', treat 'xxx' as a port.
		if strings.HasPrefix(ol, "convox.port.") && strings.HasSuffix(ol, ".protocol") {
			p := extractPort(ol)
			if p != "" {
				p, _ := strconv.Atoi(p)
				new_port := p + shift
				new_label := fmt.Sprintf("convox.port.%d.protocol", new_port)
				labels[new_label] = labels[ol]

				// Delete the old label, since we want to replace e.g. 'convox.port.443.protocol' 
				// with 'convox.port.444.protocol' entirely instead of duplicating them when we shift.
				delete(labels, ol)
			}
		}
	}
	return labels
}
