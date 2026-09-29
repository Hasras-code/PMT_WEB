FROM golang:1.26.8-alpine AS source
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM source AS build-api
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api

FROM source AS build-tools
RUN CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate && \
	CGO_ENABLED=0 go build -trimpath -o /out/migrate-storage ./cmd/migrate-storage && \
    CGO_ENABLED=0 go build -trimpath -o /out/admin ./cmd/admin && \
    CGO_ENABLED=0 go build -trimpath -o /out/jobs ./cmd/jobs

FROM alpine:3.23 AS runtime
RUN apk add --no-cache ca-certificates && addgroup -S lms && adduser -S -G lms lms && mkdir -p /data/files && chown -R lms:lms /data && chmod 700 /data/files
WORKDIR /app
USER lms

# Operational commands share the same source build but are excluded from the
# default production API image. Compose selects this target for migrations.
FROM runtime AS tools
COPY --from=build-tools --chown=lms:lms /out/migrate /out/migrate-storage /out/admin /out/jobs /app/
COPY --chown=lms:lms migrations /app/migrations

# The final/default target is the independently deployable Cloud Run API image.
FROM runtime AS api
COPY --from=build-api --chown=lms:lms /out/api /app/api
EXPOSE 8080
CMD ["/app/api"]
