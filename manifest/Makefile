all: templates

templates:
	env GOOS=linux GOARCH=amd64 go build ../changes/cmd/changed
	go-bindata -nometadata -pkg manifest -o assets.go changed
	rm changed
