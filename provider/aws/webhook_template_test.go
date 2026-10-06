package aws

import (
	"bytes"
	"encoding/json"
	"html/template"
	"path/filepath"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestWebhookForwarderProtocols(t *testing.T) {
	path := filepath.Join("templates", "resource", "webhook.tmpl")
	tpl, err := template.New("webhook").Funcs(templateHelpers()).ParseFiles(path)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, tpl.ExecuteTemplate(&buf, "resource", &structs.Resource{}))

	var v struct {
		Resources struct {
			Forwarder struct {
				Properties struct {
					Code struct {
						ZipFile struct {
							Join []json.RawMessage `json:"Fn::Join"`
						}
					}
				}
			}
		}
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &v))

	join := v.Resources.Forwarder.Properties.Code.ZipFile.Join
	require.Len(t, join, 2)

	var sep string
	var lines []string
	require.NoError(t, json.Unmarshal(join[0], &sep))
	require.NoError(t, json.Unmarshal(join[1], &lines))
	src := strings.Join(lines, sep)

	require.Contains(t, src, `const http = require("http");`)
	require.Contains(t, src, `const client = target.protocol === "http:" ? http : https;`)
	require.Contains(t, src, "client.request(")
	require.NotContains(t, src, "https.request(")
}
