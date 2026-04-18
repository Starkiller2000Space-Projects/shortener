package middlewares

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"slices"

	"github.com/Starkiller2000Space-Projects/shortener/internal/utils"
)

type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	mediaType, _, err := mime.ParseMediaType(c.w.Header().Get("Content-Type"))
	if err == nil && (mediaType == "application/json" || mediaType == "text/html") {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	return c.zw.Close()
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
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
		return err
	}
	return c.zr.Close()
}

func GzipMiddleware(next http.Handler) http.Handler {
	gzipFn := func(w http.ResponseWriter, r *http.Request) {
		ow := w // copy writer

		acceptEncoding := utils.ParseAcceptEncoding(r.Header.Get("Accept-Encoding"))
		if q, ok := acceptEncoding["gzip"]; ok && q > 0 {
			cw := newCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		if slices.Contains(utils.ParseContentEncoding(r.Header.Get("Content-Encoding")), "gzip") {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		next.ServeHTTP(ow, r)
	}
	return http.HandlerFunc(gzipFn)
}
