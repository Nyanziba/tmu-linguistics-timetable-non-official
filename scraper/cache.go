package scraper

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const defaultCacheMaxAge = 24 * time.Hour

// CachedFetch はURLの内容を取得し、ファイルキャッシュを利用する。
// キャッシュが有効期限内であればファイルから読み込み、
// 期限切れまたは存在しなければHTTPで取得してキャッシュに保存する。
func CachedFetch(url string, cacheDirectory string, maxAge time.Duration) ([]byte, error) {
	if maxAge == 0 {
		maxAge = defaultCacheMaxAge
	}

	cacheFilePath := buildCacheFilePath(cacheDirectory, url)

	if cachedData, err := readCacheIfFresh(cacheFilePath, maxAge); err == nil {
		return cachedData, nil
	}

	responseBody, err := fetchFromNetwork(url)
	if err != nil {
		return nil, err
	}

	if writeErr := writeCacheFile(cacheDirectory, cacheFilePath, responseBody); writeErr != nil {
		fmt.Fprintf(os.Stderr, "キャッシュ書き込み警告: %v\n", writeErr)
	}

	return responseBody, nil
}

func buildCacheFilePath(cacheDirectory string, url string) string {
	hash := sha256.Sum256([]byte(url))
	fileName := fmt.Sprintf("%x.html", hash[:8])
	return filepath.Join(cacheDirectory, fileName)
}

func readCacheIfFresh(cacheFilePath string, maxAge time.Duration) ([]byte, error) {
	fileInfo, err := os.Stat(cacheFilePath)
	if err != nil {
		return nil, err
	}

	if time.Since(fileInfo.ModTime()) > maxAge {
		return nil, fmt.Errorf("キャッシュ期限切れ")
	}

	return os.ReadFile(cacheFilePath)
}

func fetchFromNetwork(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%s の取得に失敗: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s がステータス %d を返しました", url, response.StatusCode)
	}

	return io.ReadAll(response.Body)
}

func writeCacheFile(cacheDirectory string, cacheFilePath string, data []byte) error {
	if err := os.MkdirAll(cacheDirectory, 0755); err != nil {
		return err
	}
	return os.WriteFile(cacheFilePath, data, 0644)
}
