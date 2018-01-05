package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"encoding/base64"

	"net/url"

	"strings"

	"io/ioutil"

	"github.com/headzoo/surf/jar"
	"github.com/headzoo/ut"
)

func TestBrowserForm(t *testing.T) {
	ts := setupTestServer(htmlForm, t)
	defer ts.Close()

	bow := &Browser{}
	bow.headers = make(http.Header, 10)
	bow.history = jar.NewMemoryHistory()

	err := bow.Open(ts.URL)
	ut.AssertNil(err)

	f, err := bow.Form("[name='default']")
	ut.AssertNil(err)

	f.Input("age", "55")
	f.Input("gender", "male")
	err = f.Click("submit2")
	ut.AssertNil(err)
	ut.AssertContains("age=55", bow.Body())
	ut.AssertContains("gender=male", bow.Body())
	ut.AssertContains("submit2=submitted2", bow.Body())
	ut.AssertContains("car=volvo", bow.Body())
}

func TestSubmitMultipart(t *testing.T) {
	ts := setupTestServer(multipartForm, t)
	defer ts.Close()

	bow := &Browser{}
	bow.headers = make(http.Header, 10)
	bow.history = jar.NewMemoryHistory()

	err := bow.Open(ts.URL)
	ut.AssertNil(err)

	f, err := bow.Form("[name='default']")
	ut.AssertNil(err)

	f.Input("comment", "my profile picture")
	imgData, err := base64.StdEncoding.DecodeString(image)
	f.File("image", "profile.png", bytes.NewBuffer(imgData))
	err = f.Submit()
	ut.AssertNil(err)
	ut.AssertContains("comment=my+profile+picture", bow.Body())
	ut.AssertContains("image=profile.png", bow.Body())
	ut.AssertContains(fmt.Sprintf("profile.png=%s", url.QueryEscape(image)), bow.Body())
}

func TestBrowserFormClickByValue(t *testing.T) {
	ts := setupTestServer(htmlFormClick, t)
	defer ts.Close()

	bow := &Browser{}
	bow.headers = make(http.Header, 10)
	bow.history = jar.NewMemoryHistory()

	err := bow.Open(ts.URL)
	ut.AssertNil(err)

	f, err := bow.Form("[name='default']")
	ut.AssertNil(err)

	f.Input("age", "55")
	f.Input("car", "saab")
	err = f.ClickByValue("submit", "submitted2")
	ut.AssertNil(err)
	ut.AssertContains("age=55", bow.Body())
	ut.AssertContains("submit=submitted2", bow.Body())
	ut.AssertContains("car=saab", bow.Body())
}

func setupTestServer(html string, t *testing.T) *httptest.Server {
	ut.Run(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprint(w, html)
		} else {
			ct := r.Header.Get("Content-Type")
			if strings.LastIndex(ct, "multipart/form-data") != -1 {

				r.ParseMultipartForm(1024 * 1024) // max 1MB ram
				values := url.Values{}
				for k, av := range r.MultipartForm.Value {
					for _, v := range av {
						values.Add(k, v)
					}
				}
				for k, af := range r.MultipartForm.File {
					for _, fh := range af {
						values.Add(k, fmt.Sprintf("%s", fh.Filename))
						f, _ := fh.Open()
						data, _ := ioutil.ReadAll(f)
						val := base64.StdEncoding.EncodeToString(data)
						values.Add(fh.Filename, val)
					}
				}
				fmt.Fprint(w, values.Encode())
			} else {
				r.ParseForm()
				fmt.Fprint(w, r.Form.Encode())
			}

		}
	}))

	return ts
}

var htmlForm = `<!doctype html>
<html>
	<head>
		<title>Echo Form</title>
	</head>
	<body>
		<form method="post" action="/" name="default">
			<input type="text" name="age" value="" />
			<input type="radio" name="gender" value="male" />
			<input type="radio" name="gender" value="female" />
			<select name="car">
				<option value="volvo">Volvo</option>
				<option value="saab">Saab</option>
				<option value="mercedes">Mercedes</option>
				<option value="audi">Audi</option>
			</select>			
			<input type="submit" name="submit1" value="submitted1" />
			<input type="submit" name="submit2" value="submitted2" />
		</form>
	</body>
</html>
`

var htmlFormClick = `<!doctype html>
<html>
	<head>
		<title>Echo Form</title>
	</head>
	<body>
		<form method="post" action="/" name="default">
			<input type="text" name="age" value="" />
			<select name="car"></select>			
			<input type="submit" name="submit" value="submitted1" />
			<input type="submit" name="submit" value="submitted2" />
		</form>
	</body>
</html>
`

var multipartForm = `<!doctype html>
<html>
	<head>
		<title>multipart form</title>
	</head>
	<body>
		<form method="post" action="/" name="default" enctype="multipart/form-data">
			<input type="text" name="comment" value="" />
			<input type="file" name="image" />
			<input type="submit" name="submit" value="submitted1" />
		</form>
	</body>
</html>
`

var image = `iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAYAAABzenr0AAACjUlEQVRYR+2Wy6oiMRCG4x0V76CIim504fu/gk/hQlS8LhQVERR15stQPZl0a3IOwtlMgXSnU/nrq0p12thoNHqqH7T4D8bWof8DeFXg+fzTJnKldK57c/7dNjsBYrGY4hcFkUwmVSqV0vMmFL7y7F1w5pIuh8fjoeLxv5y5XE61Wi1VKpUUANj9fleHw0Etl0t1Op0CSR8QJ4BkD0i321XtdltXw8wQkGq1qmq1moaYTqeuvIJ5J4B4djodHVy2xI4gQM1mUwPOZjNdOVcvOHuAQJQdAF9ji4rFojM4el4ACPo2lWwZa3zMCUAJy+Wyj5b2wZ/SFwqFf5r3lYATIJ1OB93+SsR8LhUAIpPJOJc4AUTB1Ux2pFfNavs5AW63m+IVlMxsAXssryjXy+ViT4fGTgAC73Y7vdCnEfEBmIPJp2pOAMTW67UW456rCJtj7mUM7Gq1CmUb9cAJQEYcr4vFQldAgnA1zazOdrtV+/3eq2JOAIKQ8Xw+11lJL3A1g8rebzYbNZlMAtiorM1nTgBzH8nseDxGvt9SKbZLIF3BmY/Z/wklEzMwZwFHcb1ejwxuBmJrAKVi1+s1mDIrZ/o7P0b5fF71+339PfAxAjUaDX0SjsdjdT6f3y4LbQEC0mAEHw6HOrhZkbeKvyfxZQ1r0cDQNHtGNEIAOCYSCX38DgYD/Y8Hi1osIvZVfFmLBlpo2m8O60IAsrjX66lsNqsDmz87mD22/dFAC4tKIhKAstFwnzK00PQCYP/kb9enAN5phirAJ5TvfxTtd4HQQjPq8xwCqFQqQfCvdP4rONEAAm3bQgC8v5/MXgKiibZtIQCaxaS2F3x1LMmgKWeCqREC4Nj9ROltUDTRtu0X2hs2IkarWoAAAAAASUVORK5CYII=`
