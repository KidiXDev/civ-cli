package civitai

// ResponseMetadata holds pagination and cursor information for list endpoints.
type ResponseMetadata struct {
	TotalItems  int         `json:"totalItems"`
	CurrentPage int         `json:"currentPage"`
	PageSize    int         `json:"pageSize"`
	TotalPages  int         `json:"totalPages"`
	NextPage    string      `json:"nextPage"`
	PrevPage    string      `json:"prevPage"`
	NextCursor  interface{} `json:"nextCursor"`
}

// ListResponse is a generic response for list endpoints.
type ListResponse[T any] struct {
	Items    []T              `json:"items"`
	Metadata ResponseMetadata `json:"metadata"`
}

// SearchModelsOptions contains filtering and pagination options for searching models.
type SearchModelsOptions struct {
	Query  string
	Limit  int
	Page   int
	Types  []string
	Sort   string
	Period string
	Rating int
	NSFW   bool
}
