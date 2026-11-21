FROM golang:1.26.4-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/aero ./cmd/aero

FROM alpine:3.22

RUN apk add --no-cache ca-certificates su-exec postgresql17-client \
	&& adduser -D -u 65532 -h /home/aero -s /sbin/nologin aero \
	&& mkdir -p /backups \
	&& chown aero:aero /backups

COPY --from=build /out/aero /usr/local/bin/aero
COPY docker-entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh

ENV ADDR=0.0.0.0:8080 \
	HOME=/home/aero \
	BACKUP_DIR=/backups

VOLUME /backups
EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]
