package middlewares

import (
	"compress/gzip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/requests"
	"go.uber.org/zap"
)

// writer for compressed data
type compressWriter struct {
	w           http.ResponseWriter
	zw          *gzip.Writer
	compressed  bool
	wroteHeader bool
}

// get new compressed data writer
func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w: w,
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	if c.compressed {
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

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

func (c *compressWriter) Close() error {
	if c.compressed && c.zw != nil {
		return c.zw.Close()
	}
	return nil
}

// reader for compressed data
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// get new reader for compressed data
func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return fmt.Errorf("Failed close compressed data reader: %w", err)
	}
	return c.zr.Close()
}

// middleware for
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
