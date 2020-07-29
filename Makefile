.PHONY: clean build deploy

test:
	go test ./...

build:
	env GOFLAGS="-mod=vendor" go build -o bin/ministaller cmd/ministaller/*.go

build-windows:
	env GOFLAGS="-mod=vendor" -ldflags="-H windowsgui" go build -o bin/ministaller cmd/ministaller/*.go

clean:
	rm -rf ./bin ./vendor Gopkg.lock
