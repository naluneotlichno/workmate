package archive

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type Result struct {
	Filename string
	Err      string
}

const (
	defaultHTTPTimeout             = 20 * time.Second
	archiveDirPerm     os.FileMode = 0o750
)

type ctxKey int

const (
	ctxKeyHTTPTimeout ctxKey = iota
)

func WithHTTPTimeout(parent context.Context, timeout time.Duration) context.Context {
	return context.WithValue(parent, ctxKeyHTTPTimeout, timeout)
}

func httpTimeoutFromContext(ctx context.Context) time.Duration {
	v := ctx.Value(ctxKeyHTTPTimeout)
	if d, ok := v.(time.Duration); ok && d > 0 {
		return d
	}
	return defaultHTTPTimeout
}

func BuildArchive(ctx context.Context, destZipPath string, urls []string) ([]Result, error) {
	if len(urls) == 0 {
		return nil, errors.New("no urls provided")
	}

	zipFile, zipWriter, err := prepareZip(destZipPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = zipWriter.Close() }()
	defer func() { _ = zipFile.Close() }()

	client := &http.Client{Timeout: httpTimeoutFromContext(ctx)}

	results := make([]Result, len(urls))
	usedNames := make(map[string]int, len(urls))
	for i, rawURL := range urls {
		res := processURL(ctx, client, zipWriter, rawURL, i)

		if res.Filename != "" {
			base := res.Filename
			if count, ok := usedNames[base]; ok {
				count++
				usedNames[base] = count
				ext := filepath.Ext(base)
				name := strings.TrimSuffix(base, ext)
				res.Filename = fmt.Sprintf("%s(%d)%s", name, count, ext)
			} else {
				usedNames[base] = 1
			}
		}
		results[i] = res
	}

	if err := zipWriter.Close(); err != nil {
		log.Error().Err(err).Msg("closing zip writer failed")
		return results, fmt.Errorf("close zip writer: %w", err)
	}
	if err := zipFile.Close(); err != nil {
		log.Error().Err(err).Msg("closing zip file failed")
		return results, fmt.Errorf("close zip file: %w", err)
	}
	return results, nil
}

func prepareZip(destZipPath string) (io.WriteCloser, *zip.Writer, error) {
	zipFile, err := createFile(destZipPath)
	if err != nil {
		return nil, nil, err
	}
	zipWriter := zip.NewWriter(zipFile)
	return zipFile, zipWriter, nil
}

func processURL(ctx context.Context, client *http.Client, zipWriter *zip.Writer, rawURL string, index int) Result {
	url := strings.TrimSpace(rawURL)
	filename := deriveFilename(url, index)
	result := Result{Filename: filename}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.google.com/")
	httpResponse, err := client.Do(req)
	if err != nil {
		result.Err = err.Error()
		log.Warn().Str("url", url).Err(err).Msg("http request failed")
		return result
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		if httpResponse.Body != nil {
			_ = httpResponse.Body.Close()
		}
		result.Err = fmt.Sprintf("http %d", httpResponse.StatusCode)
		log.Warn().Str("url", url).Int("status", httpResponse.StatusCode).Msg("unexpected status code")
		return result
	}

	zipEntryWriter, err := zipWriter.Create(filename)
	if err != nil {
		result.Err = err.Error()
		log.Warn().Str("url", url).Err(err).Msg("zip entry create failed")
		return result
	}

	if _, err := io.Copy(zipEntryWriter, httpResponse.Body); err != nil {
		if httpResponse.Body != nil {
			_ = httpResponse.Body.Close()
		}
		result.Err = err.Error()
		log.Warn().Str("url", url).Err(err).Msg("copy into zip failed")
		return result
	}
	if httpResponse.Body != nil {
		_ = httpResponse.Body.Close()
	}
	return result
}

func deriveFilename(rawURL string, index int) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return fmt.Sprintf("file-%d", index+1)
	}
	base := path.Base(trimmed)
	if base == "/" || base == "." || base == "" {
		return fmt.Sprintf("file-%d", index+1)
	}
	return base
}

func createFile(destinationPath string) (io.WriteCloser, error) { return openOSFile(destinationPath) }

func openOSFile(destinationPath string) (io.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(destinationPath), archiveDirPerm); err != nil {
		return nil, fmt.Errorf("ensure dir: %w", err)
	}

	outputFile, err := os.Create(destinationPath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	return outputFile, nil
}
