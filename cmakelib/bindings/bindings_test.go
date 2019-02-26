package bindings

import (
	"reflect"
	"testing"
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
	if actual := vars.Values(); !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %#v found %#v", expected, actual)
	}
	delete(expected, "CHILD")
	vars.Pop()
	if actual := vars.Values(); !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %#v found %#v", expected, actual)
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
	if actual := vars.Values(); !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %#v found %#v", expected, actual)
	}
	expected["HELLO"] = "world"
	vars.Pop()
	if actual := vars.Values(); !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %#v found %#v", expected, actual)
	}

}
