package writer

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Marshaler is the interface implemented by types that
// can marshal themselves into valid Starlark.
type Marshaler interface {
	MarshalStarlark() ([]byte, error)
}

var (
	marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
)

// Marshal returns the Starlark encoding of v.
//
// Marshal traverses the value v recursively using the following type-dependent default encodings:
//
// Boolean values are encoded as True/False.
// Strings values are encoded as quoted Starlark strings.
// Array and slice values are encoded as Starlark lists, with their contents recursively encoded.
// Nil pointer values are encoded as None.
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeValue(&buf, reflect.ValueOf(v)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeValue(b *bytes.Buffer, v reflect.Value) error {
	if !v.IsValid() {
		return writeString(b, "None")
	}
	return encodeType(b, v.Type(), v)
}

func encodeType(b *bytes.Buffer, t reflect.Type, v reflect.Value) error {
	if t.Implements(marshalerType) {
		return encodeMarshaler(b, v)
	}

	switch t.Kind() {
	case reflect.Bool:
		return encodeBool(b, v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint:
		return encodeInt(b, v)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return encodeUint(b, v)
	case reflect.Float32, reflect.Float64:
		return encodeFloat(b, v)
	case reflect.String:
		return encodeString(b, v)
	case reflect.Slice:
		return encodeSlice(b, v)
	case reflect.Array:
		return encodeArray(b, v)
	case reflect.Interface, reflect.Ptr:
		return encodeInterface(b, v)
	default:
		return fmt.Errorf("unsupported encoding type for value: %#v", v)
	}
}

func encodeBool(b *bytes.Buffer, v reflect.Value) error {
	return writeString(b, strings.Title(strconv.FormatBool(v.Bool())))
}

func encodeInt(b *bytes.Buffer, v reflect.Value) error {
	return writeString(b, strconv.FormatInt(v.Int(), 10))
}

func encodeUint(b *bytes.Buffer, v reflect.Value) error {
	return writeString(b, strconv.FormatUint(v.Uint(), 10))
}

func encodeFloat(b *bytes.Buffer, v reflect.Value) error {
	return writeString(b, strconv.FormatFloat(v.Float(), 'g', -1, 64))
}

func encodeString(b *bytes.Buffer, v reflect.Value) error {
	return writeString(b, strconv.QuoteToASCII(v.String()))
}

func encodeSlice(b *bytes.Buffer, v reflect.Value) error {
	if v.IsNil() {
		return writeString(b, "[]")
	}
	return encodeArray(b, v)
}

func encodeArray(b *bytes.Buffer, v reflect.Value) error {
	if err := b.WriteByte('['); err != nil {
		return err
	}
	n := v.Len()
	for i := 0; i < n; i++ {
		if i > 0 {
			if err := writeString(b, ", "); err != nil {
				return err
			}
		}
		if err := encodeValue(b, v.Index(i)); err != nil {
			return err
		}
	}
	return b.WriteByte(']')
}

func encodeInterface(b *bytes.Buffer, v reflect.Value) error {
	if v.IsNil() {
		return writeString(b, "None")
	}
	return encodeValue(b, v.Elem())
}

func encodeMarshaler(b *bytes.Buffer, v reflect.Value) error {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return writeString(b, "None")
	}
	m, ok := v.Interface().(Marshaler)
	if !ok {
		return writeString(b, "None")
	}
	r, err := m.MarshalStarlark()
	if err != nil {
		return err
	}
	return writeString(b, string(r))
}

func writeString(b *bytes.Buffer, value string) error {
	_, err := b.WriteString(value)
	return err
}
