# マルチステージビルドを使用
FROM golang:1.21-alpine AS builder

# 作業ディレクトリを設定
WORKDIR /app

# 依存関係をコピー
COPY go.mod go.sum ./

# 依存関係をダウンロード
RUN go mod download

# ソースコードをコピー
COPY main.go ./

# バイナリをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o google-photo-uploader main.go

# 実行用の軽量イメージを作成
FROM alpine:latest

# セキュリティとcurl、ca-certificates、rcloneをインストール
RUN apk --no-cache add \
    ca-certificates \
    curl \
    && rm -rf /var/cache/apk/*

# rcloneをインストール
RUN curl -O https://downloads.rclone.org/rclone-current-linux-amd64.zip \
    && unzip rclone-current-linux-amd64.zip \
    && mv rclone-*-linux-amd64/rclone /usr/local/bin/ \
    && rm -rf rclone-*-linux-amd64.zip rclone-*-linux-amd64

# 作業ディレクトリを設定
WORKDIR /app

# 実行バイナリをコピー
COPY --from=builder /app/google-photo-uploader .

# 設定ファイルをコピー
COPY config.yaml .

# 写真ディレクトリを作成
RUN mkdir -p /data/photos

# 設定ファイルとrclone設定用のボリュームポイントを作成
VOLUME ["/data/photos", "/config"]

# 環境変数を設定
ENV CONFIG_PATH=/app/config.yaml

# ポート（必要に応じて）
EXPOSE 8080

# 非rootユーザーを作成
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 権限設定
RUN chown -R appuser:appgroup /app /data

# 非rootユーザーに切り替え
USER appuser

# ヘルスチェック
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ps aux | grep google-photo-uploader | grep -v grep || exit 1

# アプリケーションを実行
CMD ["./google-photo-uploader"]