package repository

// SortOrder направление сортировки, общее для всех репозиториев со списками.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)
