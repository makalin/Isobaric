.PHONY: all build-release run test clean

all: build-release

build-release:
	@mkdir -p bin
	go build -ldflags="-s -w" -o bin/isobaric cli/main.go

run: build-release
	./bin/isobaric

test:
	go test -v ./...

clean:
	rm -rf bin
