package domain

// Pagination represents pagination parameters
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Sort represents sorting parameters
type Sort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "asc" or "desc"
}

// Filter represents filtering parameters
type Filter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // "eq", "neq", "gt", "lt", "gte", "lte", "like", "in"
	Value    interface{} `json:"value"`
}

// ListParams represents parameters for list operations
type ListParams struct {
	Pagination
	Sort   *Sort    `json:"sort,omitempty"`
	Filter []Filter `json:"filter,omitempty"`
}

// ListResponse represents a paginated list response
type ListResponse[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}
