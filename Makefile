# Build a normal binary for development.
all:
	go build \
		--tags "sqlite_icu" \

# Install in $GOPATH/bin.
install:
	go install \
		--tags "sqlite_icu" \

# Build distribution archive.
dist-archive:
	./tools/build

# Start euterpe after building it from source.
run:
	go run --tags "sqlite_icu" main.go -D -local-fs
