FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
ARG GOPROXY
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/data-migration

FROM alpine:3.22
WORKDIR /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/server ./server
COPY docs ./docs
COPY static ./static
CMD ["./server"]
