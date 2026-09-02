package httpx

type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	Size       int `json:"size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
