.PHONY: all templates test test-deps vendor

all: templates

fixtures:
	go install github.com/convox/rack/api/cmd/fixture
	rm api/models/fixtures/*.json
	make -C api/models/fixtures

release:
	make -C provider release VERSION=$(VERSION)
	docker build -t convox/api:$(VERSION) .
	docker push convox/api:$(VERSION)

templates:
	go get -u github.com/jteeuwen/go-bindata/...
	make -C api templates
	make -C cmd templates
	make -C manifest templates
	make -C provider templates
