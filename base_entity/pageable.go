package base_entity

type PageableDBResponse[T Entity] struct {
	Data             []T
	NumberPerPage    int64
	LastItemId       string
	Total            int64
	NoOfItemsInBatch int64
}

type PageableDBRequest struct {
	NumberPerPage int64
	LastItemId    string
}
