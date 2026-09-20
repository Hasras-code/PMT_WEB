FROM golang:1.26.8-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -trimpath -o /out/admin ./cmd/admin && \
    CGO_ENABLED=0 go build -trimpath -o /out/jobs ./cmd/jobs
FROM alpine:3.23
RUN apk add --no-cache ca-certificates && addgroup -S lms && adduser -S -G lms lms && mkdir -p /data/files && chown -R lms:lms /data && chmod 700 /data/files
WORKDIR /app
COPY --from=build /out/ /app/
COPY migrations /app/migrations
USER lms
EXPOSE 8080
CMD ["/app/api"]
