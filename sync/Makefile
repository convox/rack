all: templates

templates:
	env GOOS=linux GOARCH=amd64 go build ../changes/cmd/changed
	go-bindata -nometadata -pkg sync -o templates.go changed
	rm changed
