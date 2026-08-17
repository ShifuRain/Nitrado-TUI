FROM golang:1.25-bookworm

# Tools useful for developing a Cobra/Bubble Tea CLI: git (module fetching,
# vcs stamping), a linter, and delve for debugging.
RUN apt-get update && apt-get install -y --no-install-recommends \
        git \
        ca-certificates \
        build-essential \
    && rm -rf /var/lib/apt/lists/*

# GOTOOLCHAIN=auto lets `go install` fetch a newer toolchain on the fly when
# a tool's go.mod requires one newer than this image ships with, instead of
# failing the build every time a tool bumps its minimum Go version.
ENV GOTOOLCHAIN=auto
RUN go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest \
    && go install github.com/goreleaser/goreleaser/v2@latest

ENV CGO_ENABLED=0
WORKDIR /workspace
