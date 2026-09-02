FROM node:22-alpine AS frontend
WORKDIR /build
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /build/dist web/dist
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/mediator .

FROM alpine:3.21
RUN addgroup -g 1000 -S app && adduser -u 1000 -S app -G app \
  && apk add --no-cache su-exec
COPY --from=backend /out/mediator /usr/local/bin/mediator
COPY --from=frontend /build/dist /app/web/dist
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENV MEDIATOR_PORT=:42800 MEDIATOR_LOG_LEVEL=INFO
WORKDIR /app
RUN mkdir -p /config /downloads \
  && chown -R 1000:1000 /config /downloads /app/web \
  && chmod 755 /usr/local/bin/docker-entrypoint.sh \
  && sed -i 's/\r$//' /usr/local/bin/docker-entrypoint.sh
VOLUME ["/config", "/downloads"]
EXPOSE 42800
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -q -T 3 -t 1 -O /dev/null http://127.0.0.1:42800/health
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
