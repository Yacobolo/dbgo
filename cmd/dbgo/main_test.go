// Package main provides tests for the DBGo CLI.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	// Get the absolute path to testdata directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return filepath.Join(wd, "..", "..", "testdata")
}

func TestVersionCmd(t *testing.T) {
	err := versionCmd([]string{})
	if err != nil {
		t.Errorf("versionCmd() error = %v", err)
	}
}

func TestListCmd(t *testing.T) {
	td := testdataDir(t)
	tmpDir := t.TempDir()

	args := []string{
		"-models", filepath.Join(td, "models"),
		"-seeds", filepath.Join(td, "seeds"),
		"-macros", filepath.Join(td, "macros"),
		"-state", filepath.Join(tmpDir, "state.db"),
	}

	err := listCmd(args)
	if err != nil {
		t.Errorf("listCmd() error = %v", err)
	}
}

func TestDagCmd(t *testing.T) {
	td := testdataDir(t)
	tmpDir := t.TempDir()

	args := []string{
		"-models", filepath.Join(td, "models"),
		"-seeds", filepath.Join(td, "seeds"),
		"-macros", filepath.Join(td, "macros"),
		"-state", filepath.Join(tmpDir, "state.db"),
	}

	err := dagCmd(args)
	if err != nil {
		t.Errorf("dagCmd() error = %v", err)
	}
}

func TestSeedCmd(t *testing.T) {
	td := testdataDir(t)
	tmpDir := t.TempDir()

	args := []string{
		"-models", filepath.Join(td, "models"),
		"-seeds", filepath.Join(td, "seeds"),
		"-macros", filepath.Join(td, "macros"),
		"-state", filepath.Join(tmpDir, "state.db"),
	}

	err := seedCmd(args)
	if err != nil {
		t.Errorf("seedCmd() error = %v", err)
	}
}

func TestRunCmd(t *testing.T) {
	td := testdataDir(t)
	tmpDir := t.TempDir()

	args := []string{
		"-models", filepath.Join(td, "models"),
		"-seeds", filepath.Join(td, "seeds"),
		"-macros", filepath.Join(td, "macros"),
		"-state", filepath.Join(tmpDir, "state.db"),
		"-env", "test",
	}

	err := runCmd(args)
	if err != nil {
		t.Errorf("runCmd() error = %v", err)
	}
}

func TestRunCmd_Select(t *testing.T) {
	td := testdataDir(t)
	tmpDir := t.TempDir()

	args := []string{
		"-models", filepath.Join(td, "models"),
		"-seeds", filepath.Join(td, "seeds"),
		"-macros", filepath.Join(td, "macros"),
		"-state", filepath.Join(tmpDir, "state.db"),
		"-env", "test",
		"-select", "staging.stg_customers",
	}

	err := runCmd(args)
	if err != nil {
		t.Errorf("runCmd() with -select error = %v", err)
	}
}

func TestRunCmd_SelectWithDownstream(t *testing.T) {
	td := testdataDir(t)
	tmpDir := t.TempDir()

	// Use a persistent database to maintain state between runs
	dbPath := filepath.Join(tmpDir, "test.db")

	baseArgs := []string{
		"-models", filepath.Join(td, "models"),
		"-seeds", filepath.Join(td, "seeds"),
		"-macros", filepath.Join(td, "macros"),
		"-state", filepath.Join(tmpDir, "state.db"),
		"-database", dbPath,
		"-env", "test",
	}

	// First run all models to create the base tables
	// This is required because downstream models may depend on other models
	// that weren't selected
	err := runCmd(baseArgs)
	if err != nil {
		t.Fatalf("initial runCmd() error = %v", err)
	}

	// Now test select with downstream - re-run stg_customers and its downstream
	selectArgs := append(baseArgs, "-select", "staging.stg_customers", "-downstream")
	err = runCmd(selectArgs)
	if err != nil {
		t.Errorf("runCmd() with -select and -downstream error = %v", err)
	}
}

func TestCreateEngine_BadStatePath(t *testing.T) {
	td := testdataDir(t)

	// We need to parse flags to set global variables for createEngine
	// Use a helper to set up the environment
	modelsDir = filepath.Join(td, "models")
	seedsDir = filepath.Join(td, "seeds")
	macrosDir = filepath.Join(td, "macros")
	databasePath = ""
	statePath = "/nonexistent/path/state.db"

	_, err := createEngine()
	if err == nil {
		t.Error("createEngine() should fail with bad state path")
	}
}

func TestMain(m *testing.M) {
	// Run tests
	os.Exit(m.Run())
}
