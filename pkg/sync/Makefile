all: templates

templates:
	go get github.com/karalabe/xgo
	xgo -targets linux/amd64 -out changed ../changes/cmd/changed
	mv changed-linux-amd64 changed
	go-bindata -nometadata -pkg sync -o templates.go changed
	rm changed
