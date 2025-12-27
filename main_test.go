package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGetType(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{true, "boolean"},
		{1.23, "number"},
		{"hello", "string"},
		{[]interface{}{}, "array<unknown>"},
		{[]interface{}{1.0, 2.0}, "array<number>"},
		{[]interface{}{"a", "b"}, "array<string>"},
		{map[string]interface{}{"a": 1}, "object"},
		{nil, "unknown"},
		{struct{}{}, "unknown"},
	}

	for _, tt := range tests {
		result := getType(tt.input)
		if result != tt.expected {
			t.Errorf("getType(%v) = %v; want %v", tt.input, result, tt.expected)
		}
	}
}

func TestFinalizeOptionality(t *testing.T) {
	fields := map[string]*FieldInfo{
		"required": {count: 2},
		"optional": {count: 1},
		"withNull": {count: 2, hasNull: true},
	}
	finalizeOptionality(fields, 2)

	if fields["required"].Optional {
		t.Error("expected 'required' to be non-optional")
	}
	if !fields["optional"].Optional {
		t.Error("expected 'optional' to be optional")
	}
	if !fields["withNull"].Optional {
		t.Error("expected 'withNull' to be optional")
	}
}

func TestMergeField(t *testing.T) {
	fields := make(map[string]*FieldInfo)

	// First merge
	mergeField(fields, "a", 1.0)
	if fields["a"].Type != "number" || fields["a"].count != 1 {
		t.Errorf("first merge failed: %+v", fields["a"])
	}

	// Second merge (same type)
	mergeField(fields, "a", 2.0)
	if fields["a"].count != 2 {
		t.Errorf("second merge count failed: %d", fields["a"].count)
	}

	// Merge with null
	mergeField(fields, "b", nil)
	if !fields["b"].hasNull || fields["b"].count != 1 {
		t.Errorf("merge null failed: %+v", fields["b"])
	}
}

func TestAnalyzeJSON(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"name": "Alice", "age": 30.0},
		map[string]interface{}{"name": "Bob"},
	}

	fields := analyzeJSON(data)

	if fields["name"].Optional {
		t.Error("name should not be optional")
	}
	if !fields["age"].Optional {
		t.Error("age should be optional")
	}
	if fields["name"].Type != "string" {
		t.Errorf("name type should be string, got %s", fields["name"].Type)
	}
	if fields["age"].Type != "number" {
		t.Errorf("age type should be number, got %s", fields["age"].Type)
	}
}

func TestAnalyzeJSONArrayMerging(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{
			"tags": []interface{}{
				map[string]interface{}{"id": 1.0, "name": "tag1"},
			},
		},
		map[string]interface{}{
			"tags": []interface{}{
				map[string]interface{}{"id": 2.0, "extra": true},
			},
		},
	}

	fields := analyzeJSON(data)
	tags := fields["tags"]
	if tags == nil {
		t.Fatal("tags field missing")
	}

	// In the current implementation, if an array contains objects,
	// fieldInfo.Type becomes "" and children are merged.
	if tags.Children["id"] == nil || tags.Children["id"].Type != "number" {
		t.Errorf("tags.id type should be number")
	}
	if tags.Children["name"] == nil || !tags.Children["name"].Optional {
		t.Errorf("tags.name should be optional")
	}
	if tags.Children["extra"] == nil || !tags.Children["extra"].Optional {
		t.Errorf("tags.extra should be optional")
	}
}

func TestAnalyzeJSONNested(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"id": 1.0,
			"profile": map[string]interface{}{
				"bio": "hello",
			},
		},
	}

	fields := analyzeJSON(data)

	user := fields["user"]
	if user == nil || len(user.Children) == 0 {
		t.Fatal("user field or its children missing")
	}

	if user.Children["id"].Type != "number" {
		t.Errorf("user.id type should be number, got %s", user.Children["id"].Type)
	}

	profile := user.Children["profile"]
	if profile == nil || profile.Children["bio"].Type != "string" {
		t.Errorf("profile.bio type should be string")
	}
}

func TestAnalyzeJSONNullField(t *testing.T) {
	data := map[string]interface{}{
		"avatar": nil,
	}

	fields := analyzeJSON(data)

	if avatar, ok := fields["avatar"]; ok {
		// If it's just null, it should be "unknown" and "optional"
		if avatar.Type != "unknown" {
			t.Errorf("expected type 'unknown' for null field, got %q", avatar.Type)
		}
		if !avatar.Optional {
			t.Error("expected null field to be optional")
		}
	} else {
		t.Fatal("avatar field missing")
	}
}

func TestAnalyzeJSONNullUpgrade(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"a": nil},
		map[string]interface{}{"a": "hello"},
	}

	fields := analyzeJSON(data)
	if fields["a"].Type != "string" {
		t.Errorf("expected type 'string' after upgrade from null, got %q", fields["a"].Type)
	}
	if !fields["a"].Optional {
		t.Error("expected upgraded null field to be optional")
	}
}

func TestPrintTree(t *testing.T) {
	fields := map[string]*FieldInfo{
		"a": {Type: "string", count: 1},
		"b": {
			count: 1,
			Children: map[string]*FieldInfo{
				"c": {Type: "number", count: 1},
			},
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTree(fields, "", true)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()
	expectedLines := []string{
		"root",
		"├── a: string",
		"└── b",
		"    └── c: number",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("output missing expected line: %q\nGot:\n%s", line, output)
		}
	}
}

func TestMainIntegration(t *testing.T) {
	// Create a temporary JSON file
	content := `{"name": "test", "value": 123}`
	tmpfile, err := os.CreateTemp("", "test*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Prepare to capture stdout
	oldStdout := os.Stdout
	oldArgs := os.Args
	defer func() {
		os.Stdout = oldStdout
		os.Args = oldArgs
	}()

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"cmd", tmpfile.Name()}

	// Run main
	main()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "root") || !strings.Contains(output, "name: string") || !strings.Contains(output, "value: number") {
		t.Errorf("Integration test output mismatch:\n%s", output)
	}
}
