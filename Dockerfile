# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY app/go.mod app/go.sum ./
RUN apk add --no-cache gcc musl-dev && go mod download
COPY app/ .
RUN CGO_ENABLED=1 go build -o plant-diary .

# Run stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /app/plant-diary .
COPY --from=builder /app/templates/ ./templates/
COPY --from=builder /app/migrations/ ./migrations/
RUN mkdir -p /app/data && chown -R appuser:appuser /app
USER appuser
EXPOSE 8080
CMD ["./plant-diary"]
