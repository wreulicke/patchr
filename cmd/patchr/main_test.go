package main

import (
	"testing"

	"github.com/wreulicke/snap"
)

func TestRootCommand_Help(t *testing.T) {
	t.Parallel()

	s := snap.New()
	app := NewApp(map[string]string{})
	app.SetArgs([]string{"-h"})
	app.SetOut(s)
	err := app.Execute()
	if err != nil {
		t.Error(err)
	}

	s.Assert(t)
}

func TestRootCommand(t *testing.T) {
	t.Parallel()

	s := snap.New()
	app := NewApp(map[string]string{
		"name": "John",
	})
	app.SetArgs([]string{"-d", "testdata/resource"})
	app.SetOut(s)
	err := app.Execute()
	if err != nil {
		t.Error(err)
	}

	s.Assert(t)
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	s := snap.New()
	app := NewApp(map[string]string{})
	app.SetArgs([]string{"version"})
	app.SetOut(s)
	err := app.Execute()
	if err != nil {
		t.Error(err)
	}

	s.Assert(t)
}
