package starlark

import (
	"testing"

	"go.starlark.net/starlark"
)

func TestGoToStarlark(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantStr string
		wantErr bool
	}{
		{
			name:    "string",
			input:   "hello",
			wantStr: `"hello"`,
		},
		{
			name:    "int",
			input:   42,
			wantStr: "42",
		},
		{
			name:    "int64",
			input:   int64(123456789),
			wantStr: "123456789",
		},
		{
			name:    "float64",
			input:   3.14,
			wantStr: "3.14",
		},
		{
			name:    "bool true",
			input:   true,
			wantStr: "True",
		},
		{
			name:    "bool false",
			input:   false,
			wantStr: "False",
		},
		{
			name:    "nil",
			input:   nil,
			wantStr: "None",
		},
		{
			name:    "string slice",
			input:   []string{"a", "b", "c"},
			wantStr: `["a", "b", "c"]`,
		},
		{
			name:    "empty string slice",
			input:   []string{},
			wantStr: "[]",
		},
		{
			name:    "any slice",
			input:   []any{"x", 1, true},
			wantStr: `["x", 1, True]`,
		},
		{
			name:    "map",
			input:   map[string]any{"key": "value"},
			wantStr: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GoToStarlark(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GoToStarlark() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.wantStr {
				t.Errorf("GoToStarlark() = %v, want %v", got.String(), tt.wantStr)
			}
		})
	}
}

func TestStarlarkToGo(t *testing.T) {
	tests := []struct {
		name    string
		input   starlark.Value
		want    any
		wantErr bool
	}{
		{
			name:  "string",
			input: starlark.String("hello"),
			want:  "hello",
		},
		{
			name:  "int",
			input: starlark.MakeInt(42),
			want:  int64(42),
		},
		{
			name:  "float",
			input: starlark.Float(3.14),
			want:  3.14,
		},
		{
			name:  "bool true",
			input: starlark.Bool(true),
			want:  true,
		},
		{
			name:  "bool false",
			input: starlark.Bool(false),
			want:  false,
		},
		{
			name:  "none",
			input: starlark.None,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StarlarkToGo(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("StarlarkToGo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("StarlarkToGo() = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestTargetInfo_ToStarlark(t *testing.T) {
	target := &TargetInfo{
		Type:     "duckdb",
		Schema:   "analytics",
		Database: "mydb",
	}

	val := target.ToStarlark()
	if val == nil {
		t.Fatal("ToStarlark returned nil")
	}

	// Access fields via AttrNames
	str := val.String()
	if str == "" {
		t.Error("expected non-empty string representation")
	}
}

func TestThisInfo_ToStarlark(t *testing.T) {
	this := &ThisInfo{
		Name:   "monthly_revenue",
		Schema: "analytics",
	}

	val := this.ToStarlark()
	if val == nil {
		t.Fatal("ToStarlark returned nil")
	}

	str := val.String()
	if str == "" {
		t.Error("expected non-empty string representation")
	}
}
