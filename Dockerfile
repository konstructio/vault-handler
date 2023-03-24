FROM golang:alpine AS builder

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /build

# Download go dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy into the container
COPY . .

# Build the application
RUN go build -o vault-handler .

# Build final image using nothing but the binary
FROM alpine:3.17.2

COPY --from=builder /build/vault-handler /

# Command to run
ENTRYPOINT ["/vault-handler"]