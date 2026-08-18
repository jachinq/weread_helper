# syntax=docker/dockerfile:1

ARG ALPINE_MIRROR=https://mirrors.aliyun.com/alpine
ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn

# --- 前端 ---
FROM node:22-alpine AS web
ARG ALPINE_MIRROR
ARG NPM_REGISTRY
RUN sed -i "s#https\?://dl-cdn.alpinelinux.org/alpine#${ALPINE_MIRROR}#g" /etc/apk/repositories
WORKDIR /web
ENV COREPACK_NPM_REGISTRY=${NPM_REGISTRY} \
    npm_config_registry=${NPM_REGISTRY}
RUN corepack enable && corepack prepare pnpm@10.12.1 --activate
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --registry "${NPM_REGISTRY}"
COPY web/ ./
RUN pnpm run build

# --- 后端 ---
FROM golang:1.25-alpine AS go
ARG ALPINE_MIRROR
ARG GOPROXY
ARG GOSUMDB
RUN sed -i "s#https\?://dl-cdn.alpinelinux.org/alpine#${ALPINE_MIRROR}#g" /etc/apk/repositories
WORKDIR /src
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    GOTOOLCHAIN=local
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/weread-helper ./cmd/api

# --- 运行 ---
FROM alpine:3.21
ARG ALPINE_MIRROR
RUN sed -i "s#https\?://dl-cdn.alpinelinux.org/alpine#${ALPINE_MIRROR}#g" /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata su-exec \
    && adduser -D -H -u 65532 app
WORKDIR /app
COPY --from=go /out/weread-helper /app/weread-helper
COPY --from=web /web/dist /app/web
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && chown -R app:app /app
ENV GIN_MODE=release \
    LISTEN_ADDR=:8080 \
    DATABASE_PATH=/data/weread.db \
    WEB_DIR=/app/web \
    TZ=Asia/Shanghai
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/entrypoint.sh"]
