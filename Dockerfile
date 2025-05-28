FROM golang:1.23.0 as builder

WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /people-service

FROM alpine:latest

WORKDIR /app
COPY --from=builder /people-service /app/people-service
COPY --from=builder /app/.env /app/.env

EXPOSE 8080
CMD ["/app/people-service"]