FROM --platform=$BUILDPLATFORM golang:1.19-alpine AS build
WORKDIR /build

ADD go.mod go.sum ./
RUN go mod download

ADD . .
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o exporter -ldflags='-extldflags=-static' .

# real image
FROM scratch
WORKDIR /app
ENV PORT=8080
EXPOSE $PORT
CMD ["/app/exporter"]
COPY --from=build /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=build /build/exporter /app/exporter
