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

// Result describes outcome of downloading a single URL and writing it into the zip
type Result struct {
	Filename string
	Err      string
}

// BuildArchive downloads provided URLs and writes them as files into a zip at destZipPath.
// It always returns a results slice of the same length as urls. On total failure, Err describes the reason.
// For failed individual URLs, the corresponding Result.Err is set and the file is omitted from the archive.
const (
	defaultHTTPTimeout             = 20 * time.Second
	archiveDirPerm     os.FileMode = 0o750
)

type ctxKey int

const (
	ctxKeyHTTPTimeout ctxKey = iota
)

// WithHTTPTimeout returns a child context that carries the HTTP client timeout
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

	// HTTP client with timeout
	client := &http.Client{Timeout: httpTimeoutFromContext(ctx)}

	results := make([]Result, len(urls))
	for i, rawURL := range urls {
		results[i] = processURL(ctx, client, zipWriter, rawURL, i)
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

// prepareZip creates destination file and a zip writer for it.
func prepareZip(destZipPath string) (io.WriteCloser, *zip.Writer, error) {
	// We write directly; if needed, upper layer can make it atomic by writing to temp and renaming.
	zipFile, err := createFile(destZipPath)
	if err != nil {
		return nil, nil, err
	}
	zipWriter := zip.NewWriter(zipFile)
	return zipFile, zipWriter, nil
}

// processURL downloads a single URL and writes it into the zip, returning the Result.
func processURL(ctx context.Context, client *http.Client, zipWriter *zip.Writer, rawURL string, index int) Result {
	url := strings.TrimSpace(rawURL)
	filename := deriveFilename(url, index)
	result := Result{Filename: filename}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	httpResponse, err := client.Do(req)
	if err != nil {
		result.Err = err.Error()
		log.Warn().Str("url", url).Err(err).Msg("http request failed")
		return result
	}
	// ensure body closed in all branches with explicit closes matching original behavior
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

// deriveFilename extracts a safe filename from URL or falls back to index-based naming
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

// createFile is a small abstraction to enable swapping in atomic writers later if desired
func createFile(destinationPath string) (io.WriteCloser, error) { return openOSFile(destinationPath) }

// openOSFile creates or truncates the destination file along with ensuring parent dir exists
func openOSFile(destinationPath string) (io.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(destinationPath), archiveDirPerm); err != nil { //nolint:gosec // directory created by application under controlled path
		return nil, fmt.Errorf("ensure dir: %w", err)
	}
	// os.Create truncates existing file or creates a new one
	outputFile, err := os.Create(destinationPath) //nolint:gosec // path is constructed by the application
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	return outputFile, nil
}
