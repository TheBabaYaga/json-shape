# json-shape

A command-line tool written in Go that analyzes JSON data and generates a visual tree representation of its structure, including field types and optionality detection.

## Features

- **Automatic Schema Detection**: Analyzes JSON data to infer the structure and types of all fields
- **Optional Field Detection**: Identifies optional fields by analyzing presence across multiple objects
- **Nested Structure Support**: Handles deeply nested objects and arrays
- **Multiple Input Sources**: Supports reading from:
  - Standard input (stdin)
  - Local files
  - HTTP/HTTPS URLs
- **Tree Visualization**: Displays the JSON structure as an easy-to-read tree with types and optional markers
- **Array Merging**: Intelligently merges schemas from arrays of objects

## Installation

### From Source

```bash
git clone https://github.com/TheBabaYaga/json-shape.git
cd json-shape
go build -o json-shape
```

### Using Go Install

```bash
go install github.com/TheBabaYaga/json-shape@latest
```

## Usage

### Basic Usage

Read from a file:
```bash
json-shape path/to/file.json
```

Read from stdin:
```bash
cat file.json | json-shape
# or
echo '{"name": "test", "value": 123}' | json-shape
```

Read from a URL:
```bash
json-shape https://api.example.com/data.json
```

### Output Format

The tool outputs a tree structure showing:
- Field names
- Field types (string, number, boolean, object, array, null)
- Optional fields marked with `(optional)`
- Nested structures with proper indentation

Example output:
```
root
├── name: string
├── age: number (optional)
├── address
│   ├── street: string
│   ├── city: string
│   └── postalCode: string (optional)
└── tags
    ├── id: number
    └── name: string (optional)
```

## How It Works

1. **JSON Parsing**: The tool parses JSON data into a generic Go interface structure
2. **Field Analysis**: It recursively analyzes all fields, determining their types and tracking their presence
3. **Type Inference**: Types are inferred from the actual values:
   - `string` for text values
   - `number` for numeric values (Go represents JSON numbers as float64)
   - `boolean` for true/false values
   - `object` for nested objects
   - `array<type>` for arrays (e.g., `array<string>`, `array<number>`)
   - `null` for null values
4. **Optionality Detection**: A field is marked as optional if:
   - It appears in fewer objects than the parent object count
   - It has a null value at least once
5. **Schema Merging**: When analyzing arrays of objects, the tool merges all object schemas to create a unified structure

## Examples

### Simple Object

Input (`simple.json`):
```json
{
  "name": "Alice",
  "age": 30,
  "email": "alice@example.com"
}
```

Output:
```
root
├── age: number
├── email: string
└── name: string
```

### Array of Objects with Optional Fields

Input (`employees.json`):
```json
[
  {"name": "Alice", "age": 30, "department": "Engineering"},
  {"name": "Bob", "age": 25},
  {"name": "Charlie", "age": 35, "department": "Sales", "manager": "Alice"}
]
```

Output:
```
root
├── age: number
├── department: string (optional)
├── manager: string (optional)
└── name: string
```

### Nested Objects

Input:
```json
{
  "user": {
    "id": 1,
    "profile": {
      "bio": "Developer",
      "avatar": null
    }
  }
}
```

Output:
```
root
└── user
    ├── id: number
    └── profile
        ├── avatar: string (optional)
        └── bio: string
```

### Arrays of Objects

Input:
```json
{
  "tags": [
    {"id": 1, "name": "tag1"},
    {"id": 2, "extra": true}
  ]
}
```

Output:
```
root
└── tags
    ├── extra: boolean (optional)
    ├── id: number
    └── name: string (optional)
```

## Testing

Run the test suite:
```bash
go test -v
```

The test suite includes:
- Type detection tests
- Optionality detection tests
- Field merging tests
- Nested structure tests
- Integration tests

## Requirements

- Go 1.25.5 or later

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

Kevin T'Syen

## Acknowledgments

This tool is useful for:
- Understanding JSON API responses
- Documenting JSON data structures
- Debugging JSON parsing issues
- Generating schema documentation
- Analyzing large JSON datasets

