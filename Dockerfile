FROM golang:1.26.2-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /file-saver ./cmd

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /file-saver ./file-saver

CMD [ "./file-saver" ]