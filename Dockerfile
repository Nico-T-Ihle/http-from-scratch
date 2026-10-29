FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/httpserver

FROM alpine:3.20
WORKDIR /
COPY --from=builder /server /server
EXPOSE 8282
ENTRYPOINT ["/server"]