package requests

import (
	"strconv"
	"strings"
)

// parse Accept-encoding header into mapping <encoding : quality value>
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
