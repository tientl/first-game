# The client needs Ebitengine. On Windows it builds with plain Go; on Linux it
# needs cgo and the X11/GL dev packages, so the Linux targets here only cover
# the engine-free packages, and everything is vetted for Windows.
CORE := ./internal/sim/... ./internal/protocol/... ./internal/server/...

.PHONY: test vet build-windows build-windows-demo

test:
	go test -race -count=1 $(CORE)

vet:
	go vet $(CORE)
	GOOS=windows GOARCH=amd64 go vet ./...

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o dist/survival-game.exe ./cmd/game

# The demo: only Titanoboa and Megalodon (see internal/sim/demo.go).
build-windows-demo:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -tags demo -o dist/dino-isles-demo.exe ./cmd/game
