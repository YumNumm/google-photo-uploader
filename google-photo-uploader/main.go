package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config は設定ファイルの構造体
type Config struct {
	WatchDirectory        string   `yaml:"watch_directory"`
	SupportedExtensions   []string `yaml:"supported_extensions"`
	Rclone                RcloneConfig `yaml:"rclone"`
	Logging               LoggingConfig `yaml:"logging"`
	Upload                UploadConfig `yaml:"upload"`
}

// RcloneConfig はrclone設定の構造体
type RcloneConfig struct {
	RemoteName         string `yaml:"remote_name"`
	AlbumName          string `yaml:"album_name"`
	DeleteAfterUpload  bool   `yaml:"delete_after_upload"`
	CheckDuplicates    bool   `yaml:"check_duplicates"`
}

// LoggingConfig はログ設定の構造体
type LoggingConfig struct {
	Level       string `yaml:"level"`
	FilePath    string `yaml:"file_path"`
	MaxSize     int    `yaml:"max_size"`
	MaxBackups  int    `yaml:"max_backups"`
	MaxAge      int    `yaml:"max_age"`
}

// UploadConfig はアップロード設定の構造体
type UploadConfig struct {
	ConcurrentUploads int `yaml:"concurrent_uploads"`
	WaitTime          int `yaml:"wait_time"`
	RetryCount        int `yaml:"retry_count"`
	RetryInterval     int `yaml:"retry_interval"`
}

// PhotoUploader は写真アップロードの管理構造体
type PhotoUploader struct {
	config     *Config
	logger     *logrus.Logger
	watcher    *fsnotify.Watcher
	uploadChan chan string
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewPhotoUploader は新しいPhotoUploaderを作成
func NewPhotoUploader(configPath string) (*PhotoUploader, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗しました: %v", err)
	}

	logger := setupLogger(config.Logging)
	
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("ファイル監視の初期化に失敗しました: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PhotoUploader{
		config:     config,
		logger:     logger,
		watcher:    watcher,
		uploadChan: make(chan string, 100),
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// loadConfig は設定ファイルを読み込む
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setupLogger はログ設定を初期化
func setupLogger(config LoggingConfig) *logrus.Logger {
	logger := logrus.New()
	
	// ログレベル設定
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// ログフォーマット設定
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	return logger
}

// Start はアップローダーを開始
func (p *PhotoUploader) Start() error {
	p.logger.Info("Google Photo HEIF Uploader を開始します")
	
	// 監視ディレクトリの存在確認
	if _, err := os.Stat(p.config.WatchDirectory); os.IsNotExist(err) {
		return fmt.Errorf("監視ディレクトリが存在しません: %s", p.config.WatchDirectory)
	}

	// 監視ディレクトリを追加
	if err := p.watcher.Add(p.config.WatchDirectory); err != nil {
		return fmt.Errorf("監視ディレクトリの追加に失敗しました: %v", err)
	}

	p.logger.Infof("監視ディレクトリ: %s", p.config.WatchDirectory)

	// アップロード処理のワーカーを開始
	for i := 0; i < p.config.Upload.ConcurrentUploads; i++ {
		p.wg.Add(1)
		go p.uploadWorker(i)
	}

	// ファイル監視を開始
	p.wg.Add(1)
	go p.watchFiles()

	return nil
}

// Stop はアップローダーを停止
func (p *PhotoUploader) Stop() {
	p.logger.Info("Google Photo HEIF Uploader を停止します")
	
	p.cancel()
	p.watcher.Close()
	close(p.uploadChan)
	p.wg.Wait()
	
	p.logger.Info("Google Photo HEIF Uploader を停止しました")
}

// watchFiles はファイル監視を実行
func (p *PhotoUploader) watchFiles() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case event, ok := <-p.watcher.Events:
			if !ok {
				return
			}
			
			if event.Op&fsnotify.Create == fsnotify.Create {
				p.handleFileCreated(event.Name)
			}
			
		case err, ok := <-p.watcher.Errors:
			if !ok {
				return
			}
			p.logger.Errorf("ファイル監視エラー: %v", err)
		}
	}
}

// handleFileCreated は新しいファイルが作成された時の処理
func (p *PhotoUploader) handleFileCreated(filePath string) {
	if !p.isSupportedFile(filePath) {
		return
	}

	p.logger.Infof("新しいファイルを検出しました: %s", filePath)

	// ファイルが完全に書き込まれるまで少し待つ
	time.Sleep(time.Duration(p.config.Upload.WaitTime) * time.Second)

	// アップロードキューに追加
	select {
	case p.uploadChan <- filePath:
		p.logger.Debugf("ファイルをアップロードキューに追加しました: %s", filePath)
	case <-p.ctx.Done():
		return
	}
}

// isSupportedFile はサポートされているファイル形式かチェック
func (p *PhotoUploader) isSupportedFile(filePath string) bool {
	ext := filepath.Ext(filePath)
	for _, supportedExt := range p.config.SupportedExtensions {
		if strings.EqualFold(ext, supportedExt) {
			return true
		}
	}
	return false
}

// uploadWorker はアップロード処理を実行するワーカー
func (p *PhotoUploader) uploadWorker(workerID int) {
	defer p.wg.Done()

	p.logger.Infof("アップロードワーカー %d を開始しました", workerID)

	for {
		select {
		case <-p.ctx.Done():
			return
		case filePath, ok := <-p.uploadChan:
			if !ok {
				return
			}
			
			p.logger.Infof("ワーカー %d: ファイルをアップロード中 %s", workerID, filePath)
			
			if err := p.uploadFile(filePath); err != nil {
				p.logger.Errorf("ワーカー %d: アップロードに失敗しました %s: %v", workerID, filePath, err)
				continue
			}
			
			p.logger.Infof("ワーカー %d: アップロードが完了しました %s", workerID, filePath)
			
			// アップロード後にファイルを削除する設定の場合
			if p.config.Rclone.DeleteAfterUpload {
				if err := os.Remove(filePath); err != nil {
					p.logger.Errorf("ワーカー %d: ファイル削除に失敗しました %s: %v", workerID, filePath, err)
				} else {
					p.logger.Infof("ワーカー %d: ファイルを削除しました %s", workerID, filePath)
				}
			}
		}
	}
}

// uploadFile はrcloneを使用してファイルをアップロード
func (p *PhotoUploader) uploadFile(filePath string) error {
	// ファイルの存在確認
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("ファイルが存在しません: %s", filePath)
	}

	// rcloneコマンドの構築
	remotePath := fmt.Sprintf("%s:%s", p.config.Rclone.RemoteName, p.config.Rclone.AlbumName)
	
	args := []string{"copy", filePath, remotePath}
	
	if p.config.Rclone.CheckDuplicates {
		args = append(args, "--checksum")
	}

	// リトライ処理
	for i := 0; i < p.config.Upload.RetryCount; i++ {
		cmd := exec.Command("rclone", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			p.logger.Warnf("アップロード試行 %d/%d に失敗しました: %v", i+1, p.config.Upload.RetryCount, err)
			
			if i < p.config.Upload.RetryCount-1 {
				time.Sleep(time.Duration(p.config.Upload.RetryInterval) * time.Second)
				continue
			}
			
			return fmt.Errorf("アップロードに失敗しました（%d回試行）: %v", p.config.Upload.RetryCount, err)
		}
		
		return nil
	}

	return nil
}

func main() {
	// 設定ファイルのパスを取得
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// PhotoUploaderを作成
	uploader, err := NewPhotoUploader(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初期化に失敗しました: %v\n", err)
		os.Exit(1)
	}

	// シグナルハンドリングの設定
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// アップローダーを開始
	if err := uploader.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "開始に失敗しました: %v\n", err)
		os.Exit(1)
	}

	// シグナルを待機
	<-sigChan

	// アップローダーを停止
	uploader.Stop()
}