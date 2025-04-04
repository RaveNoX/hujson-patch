package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func constructPatch(patch []byte, deepMaps bool) ([]byte, error) {
	var patchMap map[string]interface{}
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return nil, fmt.Errorf("failed to parse patch: %w", err)
	}

	var operations []map[string]interface{}
	if deepMaps {
		processDeep("", nil, patchMap, &operations)
	} else {
		processFlat("", patchMap, &operations)
	}
	return json.Marshal(operations)
}

func processDeep(path string, original, patch interface{}, ops *[]map[string]interface{}) {
	patchMap, isPatchMap := patch.(map[string]interface{})
	if !isPatchMap {
		if !jsonValuesEqual(original, patch) {
			op := "replace"
			if original == nil {
				op = "add"
			}
			*ops = append(*ops, map[string]interface{}{
				"op":    op,
				"path":  path,
				"value": patch,
			})
		}
		return
	}

	originalMap, _ := original.(map[string]interface{})
	if originalMap == nil {
		originalMap = make(map[string]interface{})
	}

	for key, patchValue := range patchMap {
		newPath := path + "/" + escapeJSONPointer(key)
		originalValue := originalMap[key]

		if patchValue == nil {
			if originalValue != nil {
				*ops = append(*ops, map[string]interface{}{
					"op":   "remove",
					"path": newPath,
				})
			}
		} else {
			processDeep(newPath, originalValue, patchValue, ops)
		}
	}
}

func processFlat(path string, patch interface{}, ops *[]map[string]interface{}) {
	patchMap, isPatchMap := patch.(map[string]interface{})
	if !isPatchMap {
		if patch != nil {
			*ops = append(*ops, map[string]interface{}{
				"op":    "add",
				"path":  path,
				"value": patch,
			})
		}
		return
	}

	for key, value := range patchMap {
		newPath := path + "/" + escapeJSONPointer(key)
		if value == nil {
			continue
		}
		processFlat(newPath, value, ops)
	}
}

func jsonValuesEqual(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func escapeJSONPointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}
