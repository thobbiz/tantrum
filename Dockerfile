# build stage
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependency downloads separately from source changes
COPY go.mod ./
RUN go mod download

COPY . .

# CGO disabled for a fully static binary that runs on a minimal base image
RUN CGO_ENABLED=0 GOOS=linux go build -o /tantrum .

# runtime stage
FROM alpine:3.20

RUN adduser -D -u 10001 tantrum

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=builder /tantrum /usr/local/bin/tantrum

USER tantrum

EXPOSE 8080

ENTRYPOINT ["tantrum"]
