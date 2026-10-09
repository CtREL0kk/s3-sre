FROM golang:1.27-alpine AS builder
WORKDIR /src

COPY ../go.mod go.sum ./
RUN go mod download

COPY .. .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd

FROM ubuntu:noble
COPY --from=builder /out/server /usr/local/bin/server

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/server"]