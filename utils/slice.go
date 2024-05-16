package utils

func Collisions(slice1, slice2 []string) []string {
	m := make(map[string]struct{}, len(slice1))
	for _, s := range slice1 {
		m[s] = struct{}{}
	}
	var result []string
	for _, s := range slice2 {
		if _, ok := m[s]; ok {
			result = append(result, s)
		}
	}
	return result
}
