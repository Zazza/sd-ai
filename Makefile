.PHONY: setup dev build test lint tidy clean

setup:
	go mod download
	cd frontend && npm install

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

build-server:
	cd server && go build -o ../build/sd-studio-server .

test-server:
	cd server && go test ./... -v
