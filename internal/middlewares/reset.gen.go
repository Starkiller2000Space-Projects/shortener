package middlewares

// Reset resets all fields of compressWriter to their zero values.
func (r *compressWriter) Reset() {
	if r == nil {
		return
	}
	r.w = nil
	if r.zw != nil {
		r.zw = nil
	}
	r.compressed = false
	r.wroteHeader = false
}

// Reset resets all fields of compressReader to their zero values.
func (r *compressReader) Reset() {
	if r == nil {
		return
	}
	r.r = nil
	if r.zr != nil {
		r.zr = nil
	}
}

// Reset resets all fields of responseData to their zero values.
func (r *responseData) Reset() {
	if r == nil {
		return
	}
	r.status = 0
	r.size = 0
}
