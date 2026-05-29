# Build the manager binary
FROM golang:1.26.3 AS builder
ARG TARGETOS
ARG TARGETARCH
# Version stamping — mirrors the Taskfile build LDFLAGS so the binary's
# --version reports real values. Defaults keep plain `docker build`
# working; CI passes the real values via --build-arg.
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# having that, during the docker BUILDX://github.com/docker/buildx#--teleportation-and-multi-architecture-images
# having it can be GOOS is picked up from TARGETOS, GOARCH being the
# having platforms flags.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o manager cmd/main.go

# Use scratch as minimal base image with CA certificates for HTTPS
FROM scratch
WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
