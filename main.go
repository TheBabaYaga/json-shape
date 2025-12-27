package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
)

type FieldInfo struct {
	Type     string
	Optional bool
	Children map[string]*FieldInfo
	count    int
	hasNull  bool
}

func analyzeJSON(data interface{}) map[string]*FieldInfo {
	result := make(map[string]*FieldInfo)
	total := 0

	switch v := data.(type) {
	case map[string]interface{}:
		total = 1
		for key, value := range v {
			mergeField(result, key, value)
		}
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				total++
				for key, value := range itemMap {
					mergeField(result, key, value)
				}
			}
		}
	}

	finalizeOptionality(result, total)
	return result
}

func finalizeOptionality(fields map[string]*FieldInfo, parentCount int) {
	for _, field := range fields {
		if field.count < parentCount || field.hasNull {
			field.Optional = true
		}
		if len(field.Children) > 0 {
			finalizeOptionality(field.Children, field.count)
		}
	}
}

func mergeField(fields map[string]*FieldInfo, key string, value interface{}) {
	// If value is already a *FieldInfo, we are merging two trees
	if newInfo, ok := value.(*FieldInfo); ok {
		if existing, ok := fields[key]; ok {
			if (existing.Type == "" || existing.Type == "unknown" || existing.Type == "array<unknown>") &&
				(newInfo.Type != "" && newInfo.Type != "unknown" && newInfo.Type != "array<unknown>") {
				existing.Type = newInfo.Type
			}
			existing.count += newInfo.count
			if newInfo.hasNull {
				existing.hasNull = true
			}
			for k, v := range newInfo.Children {
				mergeField(existing.Children, k, v)
			}
			return
		}
		fields[key] = newInfo
		return
	}

	if existing, ok := fields[key]; ok {
		existing.count++
		if value == nil {
			existing.hasNull = true
		}

		// Upgrade type if currently unknown
		if (existing.Type == "unknown" || existing.Type == "array<unknown>") && value != nil {
			newType := getType(value)
			if newType != "unknown" && newType != "array<unknown>" {
				existing.Type = newType
			}
		}

		// If we find children in a subsequent object, merge them
		if nestedMap, ok := value.(map[string]interface{}); ok {
			childFields := analyzeJSON(nestedMap)
			for ck, cv := range childFields {
				mergeField(existing.Children, ck, cv)
			}
		} else if nestedArray, ok := value.([]interface{}); ok {
			for _, item := range nestedArray {
				if itemMap, ok := item.(map[string]interface{}); ok {
					arrayChildren := analyzeJSON(itemMap)
					for ck, cv := range arrayChildren {
						mergeField(existing.Children, ck, cv)
					}
					existing.Type = ""
				}
			}
		}
		return
	}

	// New field found
	fieldInfo := &FieldInfo{
		Type:     getType(value),
		Children: make(map[string]*FieldInfo),
		count:    1,
		hasNull:  value == nil,
	}

	if nestedMap, ok := value.(map[string]interface{}); ok {
		fieldInfo.Children = analyzeJSON(nestedMap)
		fieldInfo.Type = ""
	} else if nestedArray, ok := value.([]interface{}); ok {
		if len(nestedArray) > 0 {
			// Merge all objects in the array
			for _, item := range nestedArray {
				if itemMap, ok := item.(map[string]interface{}); ok {
					arrayChildren := analyzeJSON(itemMap)
					for ck, cv := range arrayChildren {
						mergeField(fieldInfo.Children, ck, cv)
					}
					fieldInfo.Type = ""
				}
			}
		} else {
			fieldInfo.Type = "array<unknown>"
		}
	}

	fields[key] = fieldInfo
}

func getType(value interface{}) string {
	switch v := value.(type) {
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		if len(v) == 0 {
			return "array<unknown>"
		}
		elemType := getType(v[0])
		// If it's an object, we'll handle it in analyzeJSON
		if elemType == "object" {
			return "array"
		}
		return fmt.Sprintf("array<%s>", elemType)
	case map[string]interface{}:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func printTree(fields map[string]*FieldInfo, prefix string, isRoot bool) {
	if isRoot {
		fmt.Println("root")
		prefix = ""
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, key := range keys {
		field := fields[key]
		isLastItem := i == len(keys)-1

		// Print the current field
		connector := "├── "
		if isLastItem {
			connector = "└── "
		}

		// Format the output
		if len(field.Children) > 0 {
			// Field has children (object or array of objects)
			optionalStr := ""
			if field.Optional {
				optionalStr = " (optional)"
			}
			fmt.Printf("%s%s%s%s\n", prefix, connector, key, optionalStr)
		} else {
			// Leaf field - show type
			typeStr := field.Type
			optionalStr := ""
			if field.Optional {
				optionalStr = " (optional)"
			}
			fmt.Printf("%s%s%s: %s%s\n", prefix, connector, key, typeStr, optionalStr)
		}

		// Print children if any
		if len(field.Children) > 0 {
			childPrefix := prefix
			if isLastItem {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}

			printTree(field.Children, childPrefix, false)
		}
	}
}

func main() {
	var reader io.Reader = os.Stdin
	if len(os.Args) > 1 {
		filename := os.Args[1]
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		reader = file
	}

	var jsonData interface{}
	if err := json.NewDecoder(reader).Decode(&jsonData); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	fields := analyzeJSON(jsonData)
	printTree(fields, "", true)
}
