package middlewares

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
func (r *compressReader) Reset() {
	if r == nil {
		return
	}
	r.r = nil
	if r.zr != nil {
		r.zr = nil
	}
}
func (r *responseData) Reset() {
	if r == nil {
		return
	}
	r.status = 0
	r.size = 0
}
