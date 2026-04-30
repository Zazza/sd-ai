.PHONY: dev build test lint tidy clean

dev:
	wails dev

build:
	cd frontend && npm install && cd ..
	wails build

test:
	go vet ./...
	go test ./... -v
	cd frontend && npm run build

lint:
	go vet ./...
	cd frontend && npx vue-tsc --noEmit

tidy:
	go mod tidy

clean:
	rm -rf build/
