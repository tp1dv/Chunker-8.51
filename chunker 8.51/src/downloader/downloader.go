package downloader

import (
	"acid/chunker/src/chunk"
	"acid/chunker/src/helpers"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultRetries     = 3
	defaultTimeout     = 30 * time.Second
	defaultUserAgent   = "chunker-go-downloader/1.0"
	defaultRetryBackoff = 500 * time.Millisecond
)

type Downloader struct {
	Manifest *chunk.RenderedChunks
	client   *http.Client
	root     string
	path     string
	compilePath string
	maxBytesPerSecond int64
	retries           int
	retryBackoff      time.Duration
	timeout           time.Duration
	userAgent         string
}

func NewDownloader(url string, downloadPath string, compilePath string) *Downloader {
	root := strings.TrimSpace(url)
	if root != "" && root[len(root)-1:] == "/" {
		root = root[:len(root)-1]
	}
	if root == "" {
		root = "."
	}
	if downloadPath == "" {
		downloadPath = filepath.Join(".", "downloads")
	}
	if compilePath == "" {
		compilePath = filepath.Join(".", "compiled")
	}

	return &Downloader{
		client: &http.Client{Timeout: defaultTimeout},
		root:   root,
		path:   downloadPath,
		compilePath: compilePath,
		retries: defaultRetries,
		retryBackoff: defaultRetryBackoff,
		timeout: defaultTimeout,
		userAgent: defaultUserAgent,
	}
}

func (d *Downloader) FetchManifest(manifest string, fileOutput string) (*chunk.RenderedChunks, error) {
	if strings.TrimSpace(manifest) == "" {
		return nil, errors.New("manifest path is required")
	}
	if strings.TrimSpace(d.root) == "" {
		return nil, errors.New("download root is required")
	}
	if err := os.MkdirAll(fileOutput, 0o755); err != nil {
		return nil, err
	}

	manifestURL := d.buildURL(manifest)
	resp, err := d.getWithRetry(manifestURL, 0)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest request failed: %s", resp.Status)
	}

	bytes, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	if err != nil {
		return nil, err
	}

	var result chunk.RenderedChunks
	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	if err := validateRenderedChunks(&result); err != nil {
		return nil, err
	}
	d.Manifest = &result

	manifestPath := filepath.Join(fileOutput, filepath.Base(manifest))
	if err := writeFileAtomically(manifestPath, bytes, 0o644); err != nil {
		return nil, err
	}

	return &result, nil
}

func (d *Downloader) Download() error {
	if d.Manifest == nil {
		return errors.New("manifest is not loaded")
	}
	if err := os.MkdirAll(d.path, 0o755); err != nil {
		return err
	}

	for _, file := range d.Manifest.Files {
		fmt.Printf("File::Downloading::%s::TotalSize::%d::Chunks::%d\n", file.DisplayPath, file.Size, len(file.Chunks))
		if err := d.downloadFile(d.Manifest.ID, file); err != nil {
			return err
		}
		fmt.Printf("File::Downloaded::%s::TotalSize::%d::Chunks::%d\n", file.DisplayPath, file.Size, len(file.Chunks))
	}
	return nil
}

func (d *Downloader) DownloadThreaded(size int) error {
	if d.Manifest == nil {
		return errors.New("manifest is not loaded")
	}
	if size <= 0 {
		size = 4
	}
	if err := os.MkdirAll(d.path, 0o755); err != nil {
		return err
	}

	fmt.Println("Download::Start")
	defer fmt.Println("Download::End")

	var wait sync.WaitGroup
	limiter := make(chan struct{}, size)

	for _, file := range d.Manifest.Files {
		fmt.Printf("File::Downloading::%s::TotalSize::%d::Chunks::%d\n", file.DisplayPath, file.Size, len(file.Chunks))
		limiter <- struct{}{}
		wait.Add(1)
		go func(file *chunk.File) {
			defer wait.Done()
			defer func() { <-limiter }()
			if err := d.downloadFile(d.Manifest.ID, file); err != nil {
				fmt.Printf("Chunk::Failed::%s::TotalSize::%d::Error::%s\n", file.DisplayPath, file.Size, err.Error())
				return
			}
			fmt.Printf("File::Downloaded::%s::TotalSize::%d::Chunks::%d\n", file.DisplayPath, file.Size, len(file.Chunks))
		}(file)
	}

	wait.Wait()
	return nil
}

func (d *Downloader) TestThroughput() error {
	start := time.Now()
	resp, err := d.getWithRetry("http://speedtest.tele2.net/100MB.zip", 0)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return err
	}

	duration := time.Since(start).Seconds()
	fmt.Printf("Downloaded::%d::%f::Mbps\n", n, (float64(n)*8)/(duration*1_000_000))
	d.maxBytesPerSecond = int64(float64(n) / duration)
	fmt.Printf("MaxBytesPerSecond::%d\n", d.maxBytesPerSecond)
	return nil
}

func (d *Downloader) downloadFile(buildId string, file *chunk.File) error {
	if file == nil {
		return errors.New("file is nil")
	}
	for _, chunkItem := range file.Chunks {
		if err := d.download(buildId, chunkItem); err != nil {
			return fmt.Errorf("download %s failed: %w", file.DisplayPath, err)
		}
		fmt.Printf("Chunk::Downloaded::%s::TotalSize::%d::Hash::%s\n", file.DisplayPath, chunkItem.Size, chunkItem.Hash)
	}
	return nil
}

func (d *Downloader) get(url string) (*http.Response, error) {
	return d.getWithRetry(url, 0)
}

func (d *Downloader) getWithRetry(url string, offset int64) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= d.retries; attempt++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", d.userAgent)
		req.Header.Set("Accept-Encoding", "identity")
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}

		ctx, cancel := context.WithTimeout(req.Context(), d.timeout)
		req = req.WithContext(ctx)
		resp, err := d.client.Do(req)
		cancel()
		if err == nil {
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout {
				resp.Body.Close()
				if attempt == d.retries {
					return nil, fmt.Errorf("request failed with status %s", resp.Status)
				}
				time.Sleep(d.retryBackoff * time.Duration(attempt+1))
				continue
			}
			return resp, nil
		}
		lastErr = err
		if attempt == d.retries {
			break
		}
		time.Sleep(d.retryBackoff * time.Duration(attempt+1))
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("request failed")
}

func (d *Downloader) download(buildId string, chunkItem *chunk.Chunk) error {
	if chunkItem == nil {
		return errors.New("chunk is nil")
	}
	if strings.TrimSpace(chunkItem.Hash) == "" {
		return errors.New("chunk hash is empty")
	}

	targetPath, err := d.safeJoin(buildId, chunkItem.Hash)
	if err != nil {
		return err
	}
	if data, err := os.ReadFile(targetPath); err == nil && hashMatches(data, chunkItem.Hash) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	tempPath := targetPath + ".part"
	offset := int64(0)
	if info, err := os.Stat(tempPath); err == nil && info.Size() > 0 {
		offset = info.Size()
	}

	url := d.buildURL(filepath.ToSlash(filepath.Join(buildId, chunkItem.Hash)))
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt <= d.retries; attempt++ {
		if attempt > 0 {
			time.Sleep(d.retryBackoff * time.Duration(attempt))
		}
		resp, lastErr = d.getWithRetry(url, offset)
		if lastErr != nil {
			continue
		}
		switch resp.StatusCode {
		case http.StatusOK:
			if offset > 0 {
				resp.Body.Close()
				if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
					return err
				}
				offset = 0
				continue
			}
			if err := d.persistResponse(resp, tempPath, targetPath, chunkItem.Hash, 0); err != nil {
				resp.Body.Close()
				return err
			}
			return nil
		case http.StatusPartialContent:
			if err := d.persistResponse(resp, tempPath, targetPath, chunkItem.Hash, offset); err != nil {
				resp.Body.Close()
				return err
			}
			return nil
		case http.StatusRequestedRangeNotSatisfiable:
			resp.Body.Close()
			if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			offset = 0
			continue
		case http.StatusTooManyRequests, http.StatusRequestTimeout, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			resp.Body.Close()
			if attempt == d.retries {
				return fmt.Errorf("download failed with status %s", resp.Status)
			}
			continue
		default:
			resp.Body.Close()
			return fmt.Errorf("download failed with status %s", resp.Status)
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("download failed")
}

func (d *Downloader) downloadThreaded(file *chunk.File, limiter chan struct{}, wait *sync.WaitGroup) {
	defer wait.Done()
	defer func() { <-limiter }()

	if file.Check(d.path, d.compilePath) == nil {
		fmt.Printf("File::AlreadyDownloaded::%s::TotalSize::%d\n", file.DisplayPath, file.Size)
		return
	}

	for _, chunkItem := range file.Chunks {
		if err := d.download(d.Manifest.ID, chunkItem); err != nil {
			fmt.Printf("Chunk::Failed::%s::TotalSize::%d::Hash::%s::Error::%s\n", file.DisplayPath, chunkItem.Size, chunkItem.Hash, err.Error())
			return
		}
		fmt.Printf("Chunk::Downloaded::%s::TotalSize::%d::Hash::%s\n", file.DisplayPath, chunkItem.Size, chunkItem.Hash)
	}
}

func (d *Downloader) buildURL(path string) string {
	cleanPath := strings.TrimLeft(strings.TrimSpace(path), "/")
	base := strings.TrimRight(d.root, "/")
	if cleanPath == "" {
		return base
	}
	return base + "/" + cleanPath
}

func (d *Downloader) safeJoin(parts ...string) (string, error) {
	base := filepath.Clean(d.path)
	joined := filepath.Join(append([]string{base}, parts...)...)
	rel, err := filepath.Rel(base, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing to write outside download root: %s", joined)
	}
	return joined, nil
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempFile.Name(), perm); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempFile.Name(), path)
}

func (d *Downloader) persistResponse(resp *http.Response, tempPath string, targetPath string, expectedHash string, offset int64) error {
	flag := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if offset == 0 {
		flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	file, err := os.OpenFile(tempPath, flag, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := resp.Body.Close(); err != nil {
		return err
	}

	data, err := os.ReadFile(tempPath)
	if err != nil {
		return err
	}
	if !hashMatches(data, expectedHash) {
		_ = os.Remove(tempPath)
		return fmt.Errorf("hash verification failed for %s", expectedHash)
	}

	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, targetPath)
}

func hashMatches(data []byte, expected string) bool {
	if strings.TrimSpace(expected) == "" {
		return false
	}
	return strings.EqualFold(helpers.MD5(data), strings.TrimSpace(expected))
}

func validateRenderedChunks(result *chunk.RenderedChunks) error {
	if result == nil {
		return errors.New("manifest is empty")
	}
	if strings.TrimSpace(result.ID) == "" {
		return errors.New("manifest has no valid id")
	}
	if len(result.Files) == 0 {
		return errors.New("manifest contains no files")
	}
	for i, file := range result.Files {
		if file == nil {
			return fmt.Errorf("manifest file %d is nil", i)
		}
		if strings.TrimSpace(file.DisplayPath) == "" {
			return fmt.Errorf("manifest file %d has no display path", i)
		}
		if file.Size < 0 {
			return fmt.Errorf("manifest file %d has invalid size", i)
		}
	}
	return nil
}
