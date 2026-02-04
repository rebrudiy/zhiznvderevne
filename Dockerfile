FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/blog ./cmd/app

FROM alpine:3.20
RUN adduser -D -h /app app
WORKDIR /app
COPY --from=build /bin/blog /app/blog
COPY migrations /app/migrations
COPY config /app/config
EXPOSE 8080
USER app
ENTRYPOINT ["/app/blog"]
