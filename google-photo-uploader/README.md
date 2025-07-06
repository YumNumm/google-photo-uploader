# Google Photo HEIF Uploader

Google PhotoにHEIF形式の写真を自動アップロードするGolangアプリケーションです。
ファイルシステムの監視（inotify）を利用して、新しく追加されたHEIFファイルを自動的にrcloneを使用してGoogle Photosにアップロードします。

## 機能

- 📁 **ファイル監視**: inotifyを使用してディレクトリを監視し、新しいファイルの追加を検知
- 📸 **HEIF対応**: HEIF/HEIC形式の写真ファイルを自動認識
- ☁️ **自動アップロード**: rcloneを使用してGoogle Photosに自動アップロード
- 🔄 **並列処理**: 複数のファイルを同時にアップロード
- 🔁 **リトライ機能**: アップロード失敗時の自動リトライ
- 📝 **詳細なログ**: 処理状況の詳細なログ出力
- 🐳 **Docker対応**: Dockerコンテナとして実行可能
- ⚙️ **設定可能**: YAMLファイルによる柔軟な設定

## 必要な準備

### 1. rcloneの設定

Google Photosにアクセスするためのrclone設定が必要です。

```bash
# rcloneをインストール
curl -O https://downloads.rclone.org/rclone-current-linux-amd64.zip
unzip rclone-current-linux-amd64.zip
sudo mv rclone-*-linux-amd64/rclone /usr/local/bin/

# Google Photos用のリモートを設定
rclone config

# 設定例：
# name: google-photos
# type: google photos
# client_id: (Google Cloud Consoleで取得)
# client_secret: (Google Cloud Consoleで取得)
```

### 2. Google Cloud Console設定

1. [Google Cloud Console](https://console.cloud.google.com/)にアクセス
2. 新しいプロジェクトを作成（または既存のプロジェクトを選択）
3. Google Photos Library APIを有効化
4. 認証情報を作成（OAuth 2.0クライアントID）
5. 認証情報をrclone設定で使用

## インストール・使用方法

### Docker Composeを使用する場合（推奨）

1. **リポジトリをクローン**
   ```bash
   git clone <repository-url>
   cd google-photo-uploader
   ```

2. **必要なディレクトリを作成**
   ```bash
   mkdir -p photos logs
   ```

3. **設定ファイルを編集**
   ```bash
   cp config.yaml config.yaml.backup
   nano config.yaml
   ```

4. **Docker Composeで起動**
   ```bash
   docker-compose up -d
   ```

5. **ログの確認**
   ```bash
   docker-compose logs -f google-photo-uploader
   ```

### 手動ビルドの場合

1. **依存関係のインストール**
   ```bash
   go mod download
   ```

2. **ビルド**
   ```bash
   go build -o google-photo-uploader main.go
   ```

3. **実行**
   ```bash
   ./google-photo-uploader
   ```

## 設定ファイル

`config.yaml`ファイルで動作をカスタマイズできます：

```yaml
# 監視対象ディレクトリ
watch_directory: "/data/photos"

# 処理対象ファイル拡張子
supported_extensions:
  - ".heif"
  - ".heic"
  - ".HEIF"
  - ".HEIC"

# rclone設定
rclone:
  remote_name: "google-photos"
  album_name: "Uploaded Photos"
  delete_after_upload: false
  check_duplicates: true

# ログ設定
logging:
  level: "info"
  file_path: ""

# アップロード設定
upload:
  concurrent_uploads: 2
  wait_time: 30
  retry_count: 3
  retry_interval: 5
```

## 環境変数

| 変数名 | 説明 | デフォルト値 |
|--------|------|-------------|
| `CONFIG_PATH` | 設定ファイルのパス | `config.yaml` |
| `TZ` | タイムゾーン | `Asia/Tokyo` |

## 使用方法

1. **アプリケーションを起動**
   ```bash
   docker-compose up -d
   ```

2. **写真をアップロード**
   
   監視対象ディレクトリ（デフォルト：`./photos`）にHEIF/HEICファイルをコピーまたは移動します：
   ```bash
   cp /path/to/your/photo.heic ./photos/
   ```

3. **アップロード状況を確認**
   ```bash
   docker-compose logs -f google-photo-uploader
   ```

## トラブルシューティング

### よくある問題

1. **rclone認証エラー**
   - rcloneの設定を確認してください
   - `rclone config reconnect google-photos`で再認証

2. **ファイルが検出されない**
   - 監視ディレクトリのパスを確認
   - ファイル拡張子が設定に含まれているか確認

3. **アップロードが失敗する**
   - インターネット接続を確認
   - Google Photos APIの制限を確認
   - ログでエラーメッセージを確認

### ログレベル

デバッグ情報を確認したい場合は、設定ファイルのログレベルを変更：

```yaml
logging:
  level: "debug"
```

## 開発

### 依存関係

- [fsnotify](https://github.com/fsnotify/fsnotify) - ファイルシステム監視
- [logrus](https://github.com/sirupsen/logrus) - ログ出力
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML設定ファイル解析

### ビルド・テスト

```bash
# 依存関係のインストール
go mod download

# ビルド
go build -o google-photo-uploader main.go

# テスト実行
go test ./...

# Dockerイメージをビルド
docker build -t google-photo-uploader .
```

## ライセンス

MIT License

## 貢献

プルリクエストやイシューの報告を歓迎します。

## 注意事項

- Google Photos APIには使用制限があります
- 大量のファイルをアップロードする際は、API制限に注意してください
- 重要なファイルはバックアップを取ってから使用してください
- rcloneの認証情報は適切に管理してください

## サポート

問題が発生した場合は、以下の情報を含めてイシューを作成してください：

- 使用している環境（OS、Docker版など）
- 設定ファイルの内容（認証情報は除く）
- エラーメッセージ
- 再現手順