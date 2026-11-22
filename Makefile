GODOT ?= godot
CAPTURES ?= /mnt/holding/project-nile/interactive-kickoff/captures

.PHONY: build lint test check run capture
build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -o bin/nile-sim ./cmd/nile-sim

lint:
	test -z "$$(gofmt -l cmd internal)"
	go vet ./...
	python3 -m py_compile tools/probe.py tools/godot_check.py tools/package_linux.py

test:
	go test -race ./...

check: build lint test
	python3 tools/probe.py bin/nile-sim
	python3 tools/godot_check.py --godot $(GODOT)

run: build
	$(GODOT) --path client --editor --import --quit
	$(GODOT) --path client

capture: build
	python3 tools/godot_check.py --godot $(GODOT) --capture $(CAPTURES)
