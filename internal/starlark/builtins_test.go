package starlark

import (
	"testing"

	"go.starlark.net/starlark"
)

func TestBuildConfigDict(t *testing.T) {
	dict := BuildConfigDict(
		"my_model",
		"incremental",
		"id",
		"data-team",
		"analytics",
		[]string{"finance", "metrics"},
		map[string]any{"priority": "high"},
	)

	if dict == nil {
		t.Fatal("BuildConfigDict returned nil")
	}

	d, ok := dict.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", dict)
	}

	// Check name
	nameVal, found, _ := d.Get(starlark.String("name"))
	if !found {
		t.Error("name not found in dict")
	} else if nameVal.String() != `"my_model"` {
		t.Errorf("name = %v, want \"my_model\"", nameVal)
	}

	// Check materialized
	matlVal, found, _ := d.Get(starlark.String("materialized"))
	if !found {
		t.Error("materialized not found in dict")
	} else if matlVal.String() != `"incremental"` {
		t.Errorf("materialized = %v, want \"incremental\"", matlVal)
	}

	// Check tags
	tagsVal, found, _ := d.Get(starlark.String("tags"))
	if !found {
		t.Error("tags not found in dict")
	}
	tagsList, ok := tagsVal.(*starlark.List)
	if !ok {
		t.Errorf("tags is not a list: %T", tagsVal)
	} else if tagsList.Len() != 2 {
		t.Errorf("tags length = %d, want 2", tagsList.Len())
	}

	// Check meta
	metaVal, found, _ := d.Get(starlark.String("meta"))
	if !found {
		t.Error("meta not found in dict")
	}
	metaDict, ok := metaVal.(*starlark.Dict)
	if !ok {
		t.Errorf("meta is not a dict: %T", metaVal)
	}
	priorityVal, found, _ := metaDict.Get(starlark.String("priority"))
	if !found {
		t.Error("meta.priority not found")
	} else if priorityVal.String() != `"high"` {
		t.Errorf("meta.priority = %v, want \"high\"", priorityVal)
	}
}

func TestBuildConfigDict_Empty(t *testing.T) {
	dict := BuildConfigDict("", "", "", "", "", nil, nil)

	d, ok := dict.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", dict)
	}

	// Empty config should have no keys
	if d.Len() != 0 {
		t.Errorf("expected empty dict, got %d keys", d.Len())
	}
}

func TestPredeclared(t *testing.T) {
	config := starlark.NewDict(1)
	config.SetKey(starlark.String("name"), starlark.String("test"))

	target := &TargetInfo{
		Type:     "duckdb",
		Schema:   "main",
		Database: "test.db",
	}

	this := &ThisInfo{
		Name:   "my_model",
		Schema: "analytics",
	}

	globals := Predeclared(config, "dev", target, this)

	// Check config
	if _, ok := globals["config"]; !ok {
		t.Error("config not found in globals")
	}

	// Check env
	envVal, ok := globals["env"]
	if !ok {
		t.Error("env not found in globals")
	}
	if envVal.String() != `"dev"` {
		t.Errorf("env = %v, want \"dev\"", envVal)
	}

	// Check target
	if _, ok := globals["target"]; !ok {
		t.Error("target not found in globals")
	}

	// Check this
	if _, ok := globals["this"]; !ok {
		t.Error("this not found in globals")
	}
}

func TestPredeclared_NilTarget(t *testing.T) {
	config := starlark.NewDict(0)
	globals := Predeclared(config, "prod", nil, nil)

	// Should have config and env
	if _, ok := globals["config"]; !ok {
		t.Error("config not found in globals")
	}
	if _, ok := globals["env"]; !ok {
		t.Error("env not found in globals")
	}

	// Should not have target or this
	if _, ok := globals["target"]; ok {
		t.Error("target should not be in globals when nil")
	}
	if _, ok := globals["this"]; ok {
		t.Error("this should not be in globals when nil")
	}
}
