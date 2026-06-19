.PHONY: dev build generate check clean

dev:
	wails3 dev

build:
	wails3 build

generate:
	wails3 generate bindings

test:
	cd frontend && pnpm test:e2e

check:
	go build $(shell go list ./... | grep -v '/ios$$')
	cd frontend && pnpm tsc --noEmit

clean:
	rm -rf build/bin frontend/dist
