.PHONY: all release templates

all: templates

mocks:
	go get -u github.com/vektra/mockery/.../
	rm -f mock_Provider.go
	mockery -inpkg -name Provider

release:
	make -C aws release VERSION=$(VERSION)

templates:
	make -C aws templates
