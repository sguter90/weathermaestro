FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.work and all modules
COPY go.work go.work
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY internal/ internal/

# Build
WORKDIR /app/cmd/server
RUN go build -o /weathermaestro

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /weathermaestro /weathermaestro
EXPOSE 8059
CMD ["/weathermaestro"]
