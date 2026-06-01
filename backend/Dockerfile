# ---- 构建阶段（在 builder 原生架构上交叉编译到目标架构，避免 QEMU 模拟慢）----
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/server ./cmd/server

# ---- 运行阶段 ----
FROM alpine:3.20
# ca-certificates: 连 Supabase/LiteLLM/CPA 的 TLS；tzdata: time.LoadLocation 需要时区库
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server /app/server
RUN mkdir -p /data/buffer
ENV BUFFER_DIR=/data/buffer
EXPOSE 8080
ENTRYPOINT ["/app/server"]
