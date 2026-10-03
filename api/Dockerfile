# Build Stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /main .

# Run Stage
FROM gcr.io/distroless/base
COPY --from=builder /main /main
EXPOSE 8080
ENTRYPOINT ["/main"]
