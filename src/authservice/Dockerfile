ARG BUILDPLATFORM=linux/amd64

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm@sha256:5eec47a917f0a416d8899113528371c363538875d07934a822c1fbad9a0d6346 AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src
# restore dependencies
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -ldflags="-s -w" -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /authservice .

FROM gcr.io/distroless/static

WORKDIR /src
COPY --from=builder /authservice ./server

# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single

EXPOSE 50050
ENTRYPOINT ["/src/server"]
