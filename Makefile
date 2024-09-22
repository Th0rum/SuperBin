build: generate
	rm -rf build
	@mkdir -p build
	go build -o build/gigapaste *.go

run: build
	@cd build && ./gigapaste --bind :8080

generate:
	go generate ./...

.PHONY: run generate build