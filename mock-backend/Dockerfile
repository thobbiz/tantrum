# build stage
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY main.go ./

RUN go mod init mock-backend

RUN CGO_ENABLED=0 GOOS=linux go build -o /mock-backend main.go

# runtime stage
FROM alpine:3.20

RUN adduser -D -u 10001 mock

COPY --from=builder /mock-backend /usr/local/bin/mock-backend

USER mock

ENTRYPOINT ["mock-backend"]
