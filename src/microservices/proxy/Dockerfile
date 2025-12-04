FROM golang:1.21-alpine3.19 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o proxy main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/proxy .
EXPOSE 8000
CMD ["./proxy"]
