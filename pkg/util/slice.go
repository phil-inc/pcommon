package util

import (
	"encoding/json"
	"sort"
)

// StringSliceContains checks if string slice contains given string
func StringSliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// DoesTwoSliceIntersect checks if a string slice contans at least one value of a given slice
func DoesTwoSliceIntersect(s []string, e []string) bool {
	for _, a := range s {
		for _, b := range e {
			if a == b {
				return true
			}
		}
	}
	return false
}

// GetDistinctFromStringSlice removes duplicate values from slice
func GetDistinctFromStringSlice(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

// ConvertStructSliceToMap converts slice to map
func ConvertStructSliceToMap(structArray interface{}) ([]map[string]interface{}, error) {
	data, err := json.Marshal(structArray) // Convert to a json string
	newMap := make([]map[string]interface{}, 0)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &newMap) // Convert to a map
	if err != nil {
		return nil, err
	}
	return newMap, nil
}

// IsEqualSliceString compares if two slice has same string velues
func IsEqualSliceString(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Sort2DStringSliceByIndex sorts a 2D string slice based on the specified index.
//
// Parameters:
//   - sl: The input 2D string slice to be sorted.
//   - i: The index based on which the sorting is performed.
//
// Example:
//
//	sl := [][]string{{3, "John"}, {1, "Jack"}, {2, "Rose"}}
//	i := 0
//	Returns sl = [[1, "Jack"], [2, "Rose"], [3, "John"]]
//
// Returns the sorted 2D string slice.
func Sort2DStringSliceByIndex(sl [][]string, i int) [][]string {
	sort.SliceStable(sl, func(x, y int) bool {
		return sl[x][i] < sl[y][i]
	})

	return sl
}
