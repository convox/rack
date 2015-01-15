PORT ?= 3000

.PHONY: default dev

all: dev

dev:
	@forego start

save-deps:
	godep save -r -copy=true ./...
