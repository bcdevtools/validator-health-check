package utils

func Distinct[T comparable](slice ...T) []T {
	unique := make(map[T]bool)
	for _, t := range slice {
		unique[t] = true
	}
	distinct := make([]T, 0, len(unique))
	for t := range unique {
		distinct = append(distinct, t)
	}
	return distinct
}
