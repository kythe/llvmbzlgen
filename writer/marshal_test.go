package writer

import (
	"testing"
)

type marsh struct{}

func (m marsh) MarshalStarlark() ([]byte, error) {
	return []byte("marshaled"), nil
}

func TestMarshalling(t *testing.T) {
	tests := []struct {
		v interface{}
		e string
	}{
		{1, "1"},
		{nil, "None"},
		{1.3, "1.3"},
		{true, "True"},
		{"hello, world", `"hello, world"`},
		{[]interface{}{1, true, "hello"}, "[1, True, \"hello\"]"},
		{marsh{}, "marshaled"},
	}

	for _, test := range tests {
		a, err := Marshal(test.v)
		if err != nil {
			t.Errorf("Failed to marshal %#v: %v", test.v, err)
		} else if string(a) != test.e {
			t.Errorf("Expected %#v but got %#v", test.e, string(a))
		}
	}
}
