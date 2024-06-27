all: tinychain

tinychain: $(shell find . -name '*.go')
	cd cli && go build -o tinychain
	mkdir -p build
	mv cli/tinychain build/

.PHONY: all
