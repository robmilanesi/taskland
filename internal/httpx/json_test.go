package httpx

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestWriteJSON_EncodesAndWritesStatus(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}

	tests := []struct {
		name   string
		status int
		data   any
	}{
		{"struct", http.StatusCreated, sample{Name: "milk", N: 2}},
		{"slice", http.StatusOK, []string{"a", "b", "c"}},
		{"map", http.StatusOK, map[string]int{"count": 3}},
		{"nil", http.StatusOK, nil},
		{"primitive", http.StatusOK, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			WriteJSON(rec, tt.status, tt.data)

			if rec.Code != tt.status {
				t.Errorf("status = %d, want %d", rec.Code, tt.status)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			var got any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("response body is not valid JSON: %v (body: %s)", err, rec.Body.String())
			}

			wantBytes, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("failed to marshal expected data: %v", err)
			}
			var want any
			if err := json.Unmarshal(wantBytes, &want); err != nil {
				t.Fatalf("failed to unmarshal expected data: %v", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
		})
	}
}

func TestWriteJSON_EncodeError_LogsAndStillWritesStatus(t *testing.T) {
	var logOutput strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, nil)))
	defer slog.SetDefault(prev)

	rec := httptest.NewRecorder()

	// channels cannot be marshaled to JSON, forcing Encode to fail.
	WriteJSON(rec, http.StatusOK, make(chan int))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(logOutput.String(), "failed to encode json response") {
		t.Errorf("expected log output to mention the encode failure, got %q", logOutput.String())
	}
}

func TestDecodeJSON_Happy(t *testing.T) {
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(`{"title":"Test Title"}`))

	result := DecodeJSON(rec, req, &struct {
		Title string `json:"title"`
	}{})

	if !result {
		t.Errorf("expected result to be true")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected success got %d", rec.Code)
	}
}

func TestDecodeJSON_Oversize(t *testing.T) {
	rec := httptest.NewRecorder()

	oversizedTitle := strings.Repeat("1234", 2<<20)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks",
		strings.NewReader(fmt.Sprintf(`{"title":"%s"}`, oversizedTitle)))

	expectedResult := ErrorResponse{Error: "invalid JSON body"}

	result := DecodeJSON(rec, req, &struct {
		Title string `json:"title"`
	}{})

	if result {
		t.Errorf("expected result to be false")
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected bad request 400 status, got %d", rec.Code)
	}

	var got any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body: %s)", err, rec.Body.String())
	}

	wantBytes, err := json.Marshal(expectedResult)

	if err != nil {
		t.Fatalf("failed to marshal expected data: %v", err)
	}
	var want any
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatalf("failed to unmarshal expected data: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("body = %#v, want %#v", got, want)
	}

}
