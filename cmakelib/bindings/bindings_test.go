package bindings

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStackedLookup(t *testing.T) {
	vars := New()
	vars.Set("HELLO", "WORLD")
	vars.Push()
	if actual := vars.Get("HELLO"); actual != "WORLD" {
		t.Errorf("Expected %#v found %#v", "WORLD", actual)
	}
	vars.Pop()
	if actual := vars.Get("HELLO"); actual != "WORLD" {
		t.Errorf("Expected %#v found %#v", "WORLD", actual)
	}
}

func TestStackValues(t *testing.T) {
	vars := New()
	vars.Set("HELLO", "WORLD")
	vars.Push()
	vars.Set("CHILD", "VALUE")
	expected := map[string]string{
		"HELLO": "WORLD",
		"CHILD": "VALUE",
	}
	if diff := cmp.Diff(vars.Values(), expected); diff != "" {
		t.Errorf("Unexpected diff: %#v", diff)
	}
	delete(expected, "CHILD")
	vars.Pop()
	if diff := cmp.Diff(vars.Values(), expected); diff != "" {
		t.Errorf("Unexpected diff: %#v", diff)
	}
}

func TestOverrides(t *testing.T) {
	vars := New()
	vars.Set("HELLO", "world")
	vars.Push()
	vars.Set("HELLO", "goodbye")
	expected := map[string]string{
		"HELLO": "goodbye",
	}
	if diff := cmp.Diff(vars.Values(), expected); diff != "" {
		t.Errorf("Unexpected diff: %#v", diff)
	}
	expected["HELLO"] = "world"
	vars.Pop()
	if diff := cmp.Diff(vars.Values(), expected); diff != "" {
		t.Errorf("Unexpected diff: %#v", diff)
	}

}

func TestOverridesDeletion(t *testing.T) {
	vars := New()
	vars.Set("HELLO", "world")
	vars.Push()
	vars.Set("HELLO", "")
	expected := map[string]string{}
	if diff := cmp.Diff(vars.Values(), expected); diff != "" {
		t.Errorf("Unexpected diff: %#v", diff)
	}
	expected["HELLO"] = "world"
	vars.Pop()
	if diff := cmp.Diff(vars.Values(), expected); diff != "" {
		t.Errorf("Unexpected diff: %#v", diff)
	}
}
