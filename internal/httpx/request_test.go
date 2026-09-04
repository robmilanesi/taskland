package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePagination_NoParameters(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test",
		strings.NewReader(""))

	expectedPage := 1
	expectedSize := 20
	page, size := ParsePagination(req)

	if page != expectedPage {
		t.Errorf("Expected page %d, got %d", expectedPage, page)
	}

	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}

}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name         string
		parameters   string
		expectedPage int
		expectedSize int
	}{
		{"negativePages", "page=-7", 1, 20},
		{"negativeSize", "size=-7", 1, 20},
		{"negativeAll", "page=-7&size=-35", 1, 20},
		{"positivePage", "page=15", 15, 20},
		{"positiveSize", "size=90", 1, 90},
		{"positiveAll", "page=999&size=999", 999, 999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := fmt.Sprintf("/api/v1/test?%s", tt.parameters)
			req := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(""))
			page, size := ParsePagination(req)

			if page != tt.expectedPage {
				t.Errorf("Expected page %d, got %d", tt.expectedPage, page)
			}

			if size != tt.expectedSize {
				t.Errorf("Expected size %d, got %d", tt.expectedSize, size)
			}
		})
	}
}
