VERSION=0.9.1

.PHONY: build
build:
	mkdir -p dist
	cd vedro && \
	go build -ldflags="-X 'main.version=$(VERSION)'" -o ../dist/vedrod ./cmd/vedrod

.PHONY: run
run:
	cd vedro && \
	go run ./cmd/vedrod

.PHONY: clean
clean:
	rm -rf dist/vedrod

.PHONY: install
install: build
	sudo cp dist/vedrod /usr/bin/vedrod
	sudo mkdir /var/vedra

.PHONY: uninstall
un/install:
	sudo rm /usr/bin/vedrod
	sudo rmdir /var/vedra