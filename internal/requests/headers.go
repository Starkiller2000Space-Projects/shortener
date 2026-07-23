// Package requests provides helpers for parsing HTTP headers and context values.
package requests

import (
	"strconv"
	"strings"
)

// ParseAcceptEncoding parses the Accept-Encoding header and returns a map
// of encoding names to quality values (q). Default quality is 1.0.
// Example: "gzip, deflate;q=0.9" -> map["gzip"]=1.0, map["deflate"]=0.9
func ParseAcceptEncoding(header string) map[string]float64 {
	result := make(map[string]float64)
	for _, headerPart := range strings.Split(header, ",") {
		headerPart = strings.TrimSpace(headerPart)
		if headerPart == "" {
			continue
		}
		var encoding string
		var parameters []string
		semicolonPosition := strings.Index(headerPart, ";")
		if semicolonPosition == -1 {
			encoding = headerPart
			parameters = make([]string, 0)
		} else {
			encoding = strings.TrimSpace(headerPart[:semicolonPosition])
			parameters = strings.Split(headerPart[semicolonPosition+1:], ";")
		}
		result[encoding] = 1.0
		for _, param := range parameters {
			param = strings.TrimSpace(param)
			if strings.HasPrefix(param, "q=") {
				q, err := strconv.ParseFloat(param[2:], 64)
				if err == nil {
					result[encoding] = q
				}
				break
			}
		}
	}
	return result
}

// ParseContentEncoding parses the Content-Encoding header and returns a slice
// of encoding names (split by commas and trimmed).
// Example: "gzip, br" -> ["gzip", "br"]
func ParseContentEncoding(header string) []string {
	result := make([]string, 0)
	for _, headerPart := range strings.Split(header, ",") {
		trimmedPart := strings.TrimSpace(headerPart)
		if trimmedPart == "" {
			continue
		}
		result = append(result, trimmedPart)
	}
	return result
}
