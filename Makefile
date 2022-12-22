GIT_HASH:=$(shell git rev-parse --short HEAD)
DIRTY:=$(shell test -z "`git status --porcelain`" || echo "-dirty")
VERSION:=$(GIT_HASH)$(DIRTY)
TIME:=$(shell date -u -Iseconds)

BIN:=nfc-player

.PHONY: dev pi deps
dev: deps
	go build -ldflags "-X main.buildVersion=$(VERSION) -X main.buildTime=$(TIME)" -o $(BIN) .

deps:
	go mod download

pi: deps
	GOOS=linux GOARCH=arm GOARM=5 go build -o $(BIN) -tags=pi -ldflags "-X main.buildVersion=$(VERSION) -X main.buildTime=$(TIME)" .
