.PHONY: all release templates

all: templates

mocks:
	rm mock_Provider.go
	mockery -inpkg -name Provider

release:
	make -C aws release VERSION=$(VERSION)

templates:
	make -C aws templates
