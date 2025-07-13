package jqyaml

import (
	"encoding/json"
	"reflect"

	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// createProtojsonMarshaler creates a new protojsonMarshaler with default options
func createProtojsonMarshaler() InputMarshaler {
	return &protojsonMarshaler{
		protojsonOptions: protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		},
	}
}

// protojsonMarshaler implements InputMarshaler using protojson for Protocol Buffer messages
type protojsonMarshaler struct {
	encodeOptions    []yaml.EncodeOption
	protojsonOptions protojson.MarshalOptions
}

// Marshal converts values to gojq-compatible types, using protojson for proto.Message types
func (m *protojsonMarshaler) Marshal(v interface{}) (interface{}, error) {
	// Handle nil
	if v == nil {
		return nil, nil
	}

	// Handle proto.Message
	if msg, ok := v.(proto.Message); ok {
		b, err := m.protojsonOptions.Marshal(msg)
		if err != nil {
			return nil, err
		}
		var result interface{}
		if err := json.Unmarshal(b, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	// Handle slices that might contain proto.Message
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice:
		if rv.IsNil() {
			return nil, nil
		}
		result := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i).Interface()
			converted, err := m.Marshal(elem)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil

	case reflect.Map:
		if rv.IsNil() {
			return nil, nil
		}
		result := make(map[string]interface{})
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key()
			// Map keys must be strings for JSON
			keyStr, ok := key.Interface().(string)
			if !ok {
				// Try to convert to string
				keyStr = key.String()
			}

			value := iter.Value().Interface()
			converted, err := m.Marshal(value)
			if err != nil {
				return nil, err
			}
			result[keyStr] = converted
		}
		return result, nil

	case reflect.Ptr:
		if rv.IsNil() {
			return nil, nil
		}
		return m.Marshal(rv.Elem().Interface())
	}

	// Handle map[string]interface{} explicitly (common case)
	if mapVal, ok := v.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		for k, val := range mapVal {
			converted, err := m.Marshal(val)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	}

	// Fall back to default marshaling for other types
	return (&defaultInputMarshaler{encodeOptions: m.encodeOptions}).Marshal(v)
}

// WithProtojsonInput creates an InputMarshaler that uses protojson for Protocol Buffer messages
// This handles proto.Message types and their slices/maps correctly
func WithProtojsonInput() Option {
	return WithInputMarshaler(&protojsonMarshaler{
		protojsonOptions: protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: false,
		},
	})
}

// WithProtojsonInputOptions creates an InputMarshaler with custom protojson options
func WithProtojsonInputOptions(opts protojson.MarshalOptions) Option {
	return WithInputMarshaler(&protojsonMarshaler{
		protojsonOptions: opts,
	})
}
