# Build the sample app
FROM golang:1.17 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

#copy all go files
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o vault-go-client ./cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#FROM registry.access.redhat.com/rhel7-atomic:latest
FROM registry.access.redhat.com/ubi8/ubi
WORKDIR /

# TODO:change to use volume instead
RUN groupadd -r secretless \
             -g 777 && \
    useradd -c "secretless runner account" \
            -g secretless \
            -u 777 \
            -m \
            -r \
            secretless && \
    mkdir -p /etc/conjur/ssl && \
    mkdir -p /run/conjur && \
    # Use GID of 0 since that is what OpenShift will want to be able to read things
    chown secretless:0 /etc/conjur/ssl \
                       /run/conjur && \
    # We need open group permissions in these directories since OpenShift won't
    # match our UID when we try to write files to them
    chmod 770 /etc/conjur/ssl \
              /run/conjur

COPY --from=builder /workspace/vault-go-client .
USER 65532:65532

EXPOSE 8080
ENTRYPOINT ["/vault-go-client"]
