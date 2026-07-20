package middlewares

import (
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseDataReset(t *testing.T) {
	rd := &responseData{status: 200, size: 1024}
	rd.Reset()
	assert.Equal(t, 0, rd.status)
	assert.Equal(t, 0, rd.size)

	var rd2 *responseData
	rd2.Reset() // no panic
}

func TestCompressReaderReset(t *testing.T) {
	zr := new(gzip.Reader)
	cr := &compressReader{zr: zr}
	cr.Reset()
	assert.Nil(t, cr.zr)

	var rd2 *compressReader
	rd2.Reset() // no panic
}

func TestCompressWriterReset(t *testing.T) {
	zw := new(gzip.Writer)
	cw := &compressWriter{zw: zw, compressed: true, wroteHeader: true}
	cw.Reset()
	assert.Nil(t, cw.zw)
	assert.Equal(t, false, cw.compressed)
	assert.Equal(t, false, cw.wroteHeader)

	var rd2 *compressWriter
	rd2.Reset() // no panic
}
