VERSION=0.9

build:
	cd vedro && go build -ldflags="-X 'main.version=$(VERSION)'" -o ../vedrod ./cmd/vedrod

run:
	cd vedro && go run ./cmd/vedrod

clean:
	rm -f vedrod