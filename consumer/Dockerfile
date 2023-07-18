FROM golang:1.20 AS builder

RUN go env -w GOPROXY=direct

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download


COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app

FROM alpine:3.14
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/app .

CMD ["./app"]