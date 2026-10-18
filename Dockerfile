# Multi-stage Dockerfile for tf-move-gen
FROM --platform=$BUILDPLATFORM golang:alpine AS builder
ENV GOTOOLCHAIN=auto

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /tf-move-gen ./cmd/tf-move-gen

# Final distroless nonroot image
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=builder /tf-move-gen /usr/local/bin/tf-move-gen

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/tf-move-gen"]
CMD ["--help"]
