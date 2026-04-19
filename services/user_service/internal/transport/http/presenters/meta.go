package presenters

type PaginationRepr struct {
	Current int `json:"current"`
	Total   int `json:"total"`
	Limit   int `json:"limit"`
}

type FilterRepr struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type SortRepr struct {
	Direction string `json:"direction"`
	By        string `json:"by"`
}

type Pagination struct {
	Total    int64 `json:"total"`
	Page     int64 `json:"page"`
	PageSize int64 `json:"page_size"`
}
