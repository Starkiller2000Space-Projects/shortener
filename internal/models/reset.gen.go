package models

func (r *AuditData) Reset() {
	if r == nil {
		return
	}
	r.Action = ""
	r.URL = ""
	r.UserID = ""
}
func (r *AuditEvent) Reset() {
	if r == nil {
		return
	}
	r.Timestamp = 0
	r.Action = ""
	r.UserID = ""
	r.URL = ""
}
func (r *ShortenRequest) Reset() {
	if r == nil {
		return
	}
	r.URL = ""
}
func (r *ShortenResponse) Reset() {
	if r == nil {
		return
	}
	r.Result = ""
}
func (r *BatchShortenRequest) Reset() {
	if r == nil {
		return
	}
	r.CorrelationID = ""
	r.OriginalURL = ""
}
func (r *BatchShortenResponse) Reset() {
	if r == nil {
		return
	}
	r.CorrelationID = ""
	r.ShortURL = ""
}
func (r *UserURL) Reset() {
	if r == nil {
		return
	}
	r.ShortURL = ""
	r.OriginalURL = ""
}
