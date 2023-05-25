package util

func FlattenMap(input map[string]interface{}) map[string]interface{} {
	flattenedMap := make(map[string]interface{})
	FlattenMapWithPrefix(input, flattenedMap, "")
	return flattenedMap
}

func FlattenMapWithPrefix(input map[string]interface{}, result map[string]interface{}, prefix string) {
	for key, value := range input {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		switch t := value.(type) {
		case map[string]interface{}:
			FlattenMapWithPrefix(t, result, newKey)
		default:
			result[newKey] = t
		}
	}
}
