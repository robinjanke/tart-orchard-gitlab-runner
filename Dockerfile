FROM golang:1.25.8-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gitlab-orchard-executor ./cmd/gitlab-orchard-executor

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gitlab-orchard-executor /usr/local/bin/gitlab-orchard-executor
ENTRYPOINT ["/usr/local/bin/gitlab-orchard-executor"]
