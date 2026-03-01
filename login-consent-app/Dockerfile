# syntax=docker/dockerfile:1

FROM golang:1.23 AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /out/server /server
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/server"]