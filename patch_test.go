package patchr

import (
	"os"
	"testing"

	"github.com/wreulicke/snap"
)

func TestPatch_Replace(t *testing.T) {
	t.Parallel()
	p := NewPatcher("//", map[string]string{})
	s := snap.New()
	f, err := os.Open("testdata/replace.go")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = p.Apply(s, f, map[string]string{
		"Name": "patchr",
	})
	if err != nil {
		t.Error(err)
	}

	s.Assert(t)
}

func TestPatch_Add(t *testing.T) {
	t.Parallel()
	p := NewPatcher("//", map[string]string{})
	s := snap.New()
	f, err := os.Open("testdata/add.go")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = p.Apply(s, f, map[string]string{
		"Name": "John",
	})
	if err != nil {
		t.Error(err)
	}

	s.Assert(t)
}

func TestPatch_Cut(t *testing.T) {
	t.Parallel()
	p := NewPatcher("//", map[string]string{})
	s := snap.New()
	f, err := os.Open("testdata/cut.go")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = p.Apply(s, f, nil)
	if err != nil {
		t.Error(err)
	}

	s.Assert(t)
}

func TestPatch_Template(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data map[string]bool
	}{
		{"Enabled", map[string]bool{"Enabled": true}},
		{"Disabled", map[string]bool{"Enabled": false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewPatcher("//", map[string]string{})
			s := snap.New()
			f, err := os.Open("testdata/template.go")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			err = p.Apply(s, f, tt.data)
			if err != nil {
				t.Error(err)
			}

			s.Assert(t)
		})
	}
}

func TestPatch_Remove(t *testing.T) {
	t.Parallel()
	p := NewPatcher("//", map[string]string{})
	s := snap.New()
	f, err := os.Open("testdata/remove.go")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = p.Apply(s, f, nil)
	if err != nil {
		t.Error(err)
	}

	s.Assert(t)
}
