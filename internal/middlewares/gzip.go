// Package middlewares provides HTTP middleware for logging, auth, gzip, and audit.
package middlewares

import (
	"compress/gzip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/pool"
	"github.com/max-marek-projects/shortener/internal/requests"
	"go.uber.org/zap"
)

// generate:reset
//
// compressWriter is a wrapper around http.ResponseWriter that compresses
// response data using gzip if the content type is JSON or HTML.
type compressWriter struct {
	w           http.ResponseWriter
	zw          *gzip.Writer
	compressed  bool
	wroteHeader bool
}

// newCompressWriter creates a new compressWriter (without gzip writer yet).
// The writer is created on demand when WriteHeader is called.
func newCompressWriter(w http.ResponseWriter) *compressWriter {
	cw := compressWriterPool.Get()
	cw.w = w
	return cw
}

// Header returns the header map that will be sent by the underlying ResponseWriter.
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write writes the data to the response. If compression is enabled, it writes to the gzip writer.
// Otherwise, it writes directly to the underlying writer.
func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	if c.compressed {
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

// WriteHeader sends the HTTP status code and, if the content type is JSON or HTML,
// enables gzip compression and sets the Content-Encoding header.
func (c *compressWriter) WriteHeader(statusCode int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true
	mediaType, _, err := mime.ParseMediaType(c.w.Header().Get("Content-Type"))
	if err == nil && (mediaType == "application/json" || mediaType == "text/html") {
		c.w.Header().Set("Content-Encoding", "gzip")
		c.compressed = true
		c.zw = gzip.NewWriter(c.w)
	}
	c.w.WriteHeader(statusCode)
}

// Close closes gzip writer and returns it to pool.
func (c *compressWriter) Close() error {
	if c.compressed && c.zw != nil {
		err := c.zw.Close()
		c.zw = nil
		c.w = nil
		compressWriterPool.Put(c)
		return err
	}
	c.w = nil
	compressWriterPool.Put(c)
	return nil
}

// generate:reset
//
// compressReader wraps an io.ReadCloser and decompresses gzip-encoded data.
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// newCompressReader creates a new gzip reader from the given ReadCloser.
// Returns an error if the reader cannot be initialized.
func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}
	cr := compressReaderPool.Get()
	cr.r = r
	cr.zr = zr
	return cr, nil
}

// Read reads decompressed data from the gzip reader.
func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close closes gzip reader and returns it to pool.
func (c *compressReader) Close() error {
	err := c.zr.Close()
	if err != nil {
		return fmt.Errorf("failed to close gzip reader: %w", err)
	}
	err = c.r.Close()
	if err != nil {
		return fmt.Errorf("failed to close request: %w", err)
	}
	c.r = nil
	c.zr = nil
	compressReaderPool.Put(c)
	return nil
}

// compressWriterPool reuses compressWriter objects to reduce allocations.
var compressWriterPool = pool.NewPool(func() *compressWriter { return &compressWriter{} })

// compressReaderPool reuses compressReader objects to reduce allocations.
var compressReaderPool = pool.NewPool(func() *compressReader { return &compressReader{} })

// GzipMiddleware returns a middleware that handles gzip compression for responses
// and decompresses gzip-encoded request bodies.
// It checks the Accept-Encoding header to compress responses when supported.
// It checks the Content-Encoding header to decompress request bodies.
// The middleware uses sync.Pool to reuse compressWriter and compressReader objects.
func GzipMiddleware(next http.Handler) http.Handler {
	gzipFn := func(w http.ResponseWriter, r *http.Request) {
		ow := w // copy writer to save original value

		acceptEncoding := requests.ParseAcceptEncoding(r.Header.Get("Accept-Encoding"))
		if q, ok := acceptEncoding["gzip"]; ok && q > 0 {
			cw := newCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		if slices.Contains(requests.ParseContentEncoding(r.Header.Get("Content-Encoding")), "gzip") {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				logger.Log.Error("Failed to create compress data reader", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		next.ServeHTTP(ow, r)
	}
	return http.HandlerFunc(gzipFn)
}
