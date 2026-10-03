# Build Stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY sim/ ./sim
RUN go build -o /main ./sim/main.go

# Run Stage
FROM gcr.io/distroless/base
COPY --from=builder /main /main
EXPOSE 8090
ENTRYPOINT ["/main"]
