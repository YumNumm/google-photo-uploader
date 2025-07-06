#!/bin/bash

# Google Photo HEIF Uploader セットアップスクリプト

set -e

echo "🚀 Google Photo HEIF Uploader セットアップを開始します"

# 必要なディレクトリを作成
echo "📁 必要なディレクトリを作成中..."
mkdir -p photos
mkdir -p logs
mkdir -p config

echo "📋 設定ファイルのサンプルを作成中..."
if [ ! -f "config.local.yaml" ]; then
    cp config.yaml config.local.yaml
    echo "   config.local.yaml を作成しました（本番用設定ファイル）"
fi

echo "🔧 実行権限を設定中..."
chmod +x setup.sh

echo "📝 .gitignoreファイルを作成中..."
if [ ! -f ".gitignore" ]; then
    cat > .gitignore << 'EOF'
# Binaries
google-photo-uploader
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Application specific
photos/
logs/
config.local.yaml
rclone.conf

# Docker
.dockerignore
EOF
    echo "   .gitignoreファイルを作成しました"
fi

echo "📦 Dockerイメージをビルド中..."
docker-compose build

echo "✅ セットアップが完了しました！"
echo ""
echo "📋 次の手順:"
echo "1. rcloneの設定を行ってください:"
echo "   rclone config"
echo ""
echo "2. 設定ファイルを編集してください:"
echo "   nano config.local.yaml"
echo ""
echo "3. アプリケーションを起動してください:"
echo "   docker-compose up -d"
echo ""
echo "4. ログを確認してください:"
echo "   docker-compose logs -f google-photo-uploader"
echo ""
echo "5. 写真をアップロードしてください:"
echo "   cp /path/to/your/photo.heic ./photos/"