package writer

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEmptyMacro(t *testing.T) {
	var b strings.Builder
	writer := NewStarlarkWriter(&b)
	if err := writer.BeginMacro("hello_world"); err != nil {
		t.Fatal("Unexpected error writing macro: ", err)
	}
	if err := writer.EndMacro(); err != nil {
		t.Fatal("Unpexpected error ending macro: ", err)
	}
	if diff := cmp.Diff("def hello_world(ctx):\n    return ctx\n", b.String()); diff != "" {
		t.Error("Unexpected writer output:\n", diff)
	}
}

func TestDirectoryBuffering(t *testing.T) {
	var b strings.Builder
	writer := NewStarlarkWriter(&b)
	if err := writer.BeginMacro("hello_world"); err != nil {
		t.Fatal("Unexpected error writing macro: ", err)
	}
	for _, path := range []string{"a", "b", "c"} {
		if err := writer.PushDirectory(path); err != nil {
			t.Fatal("Unpexpected error entering directory: ", err)
		}
	}
	for _, path := range []string{"c", "b", "a"} {
		if p, err := writer.PopDirectory(); err != nil {
			t.Fatal("Unpexpected error exiting directory: ", err)
		} else if diff := cmp.Diff(path, p); diff != "" {
			t.Error("Unpexpected directory path:\n", diff)
		}
	}
	if err := writer.EndMacro(); err != nil {
		t.Fatal("Unpexpected error ending macro: ", err)
	}
	if diff := cmp.Diff("def hello_world(ctx):\n    return ctx\n", b.String()); diff != "" {
		t.Error("Unexpected writer output:\n", diff)
	}
}

func TestCommandWriting(t *testing.T) {
	var b strings.Builder
	writer := NewStarlarkWriter(&b)
	if err := writer.BeginMacro("hello_world"); err != nil {
		t.Fatal("Unexpected error writing macro: ", err)
	}
	if err := writer.PushDirectory("this/is/a/path"); err != nil {
		t.Fatal("Unpexpected error entering directory: ", err)
	}
	if err := writer.WriteCommand("run", "with", "args"); err != nil {
		t.Fatal("Unpexected error writing command: ", err)
	}
	if _, err := writer.PopDirectory(); err != nil {
		t.Fatal("Unpexpected error exiting directory: ", err)
	}
	if err := writer.EndMacro(); err != nil {
		t.Fatal("Unpexpected error ending macro: ", err)
	}
	expected := "def hello_world(ctx):\n" +
		"    ctx = ctx.push_directory(ctx, \"this/is/a/path\")\n" +
		"    ctx.run(ctx, \"with\", \"args\")\n" +
		"    ctx = ctx.pop_directory(ctx)\n" +
		"    return ctx\n"
	if diff := cmp.Diff(expected, b.String()); diff != "" {
		t.Error("Unexpected writer output:\n", diff)
	}
}

func TestInvalidMacroName(t *testing.T) {
	var b strings.Builder
	writer := NewStarlarkWriter(&b)
	if err := writer.BeginMacro("spaces are bad"); err == nil {
		t.Error("Invalid name accepted")
	}
}

func TestInvalidCommandName(t *testing.T) {
	var b strings.Builder
	writer := NewStarlarkWriter(&b)
	if err := writer.BeginMacro("hello_world"); err != nil {
		t.Fatal("Unexpected error writing macro: ", err)
	}
	if err := writer.WriteCommand("space are bad"); err == nil {
		t.Error("Invalid command name accepted")
	}
}

func TestReservedWord(t *testing.T) {
	var b strings.Builder
	writer := NewStarlarkWriter(&b)
	if err := writer.BeginMacro("return"); err != nil {
		t.Fatal("Unexpected error writing macro: ", err)
	}
	if err := writer.EndMacro(); err != nil {
		t.Fatal("Unpexpected error ending macro: ", err)
	}
	const expected = "def return_(ctx):\n    return ctx\n"
	if diff := cmp.Diff(expected, b.String()); diff != "" {
		t.Error("Unexpected writer output:\n", diff)
	}
}
