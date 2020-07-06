# Build the manager binary
FROM golang:1.14.0 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/manager/main.go cmd/manager/main.go
COPY cmd/webhook/main.go cmd/webhook/main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager cmd/manager/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o webhook cmd/webhook/main.go

# Changed base image to cftiny for OSL compliance
FROM gcr.io/paketo-buildpacks/run:tiny-cnb
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/webhook .

# Set the UID and GID to be the values for the "nonroot" tiny user
# This will allow the image to run on platforms that deny root access
USER 65532:65532
