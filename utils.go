package main

func KeysFromMap[K comparable, V int | any](m map[K]V) []K {
	keys := make([]K, len(m))

	i := 0

	for k := range m {
		keys[i] = k
		i++
	}

	return keys
}
