vendor:
	go get -u github.com/jteeuwen/go-bindata/...
	env GOOS=linux GOARCH=amd64 go build ../changes/cmd/changed
	go-bindata -o assets.go -pkg manifest changed
	rm changed
