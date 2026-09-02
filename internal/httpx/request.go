package httpx

import (
	"net/http"
	"strconv"
)

func ParsePagination(r *http.Request) (page, size int) {
	page = 1
	size = 20

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if s := r.URL.Query().Get("size"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil && parsed > 0 {
			size = parsed
		}
	}

	return page, size
}
