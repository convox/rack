PORT ?= 3000

.PHONY: default dev

default: dev

dev:
	@gin -p $(PORT) -b convox
