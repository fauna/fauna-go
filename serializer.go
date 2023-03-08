package fauna

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

const (
	fieldTag   = "fauna"
	dateFormat = "2006-01-02"
	timeFormat = "2006-01-02T15:04:05.999999999Z"
)

type typeTag string

const (
	typeTagInt    typeTag = "@int"
	typeTagLong   typeTag = "@long"
	typeTagDouble typeTag = "@double"
	typeTagDate   typeTag = "@date"
	typeTagTime   typeTag = "@time"
	typeTagDoc    typeTag = "@doc"
	typeTagMod    typeTag = "@mod"
	typeTagObject typeTag = "@object"
)

type DocumentReference struct {
	CollectionName string
	RefID          string
}

type Module struct {
	Name string
}

type InvalidTypeError struct {
	Message string
}

func (e *InvalidTypeError) Error() string {
	return "fauna: InvalidType(" + e.Message + ")"
}

func unmarshal(body []byte, into interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(body))

	var bodyMap interface{}
	if err := dec.Decode(&bodyMap); err != nil {
		return err
	}

	return decode(bodyMap, into)
}

func decode(body, into interface{}) error {
	cfg := &mapstructure.DecoderConfig{
		TagName:              fieldTag,
		DecodeHook:           decodeHook,
		Result:               into,
		Metadata:             nil,
		WeaklyTypedInput:     true,
		IgnoreUntaggedFields: true,
	}

	if decoder, err := mapstructure.NewDecoder(cfg); err != nil {
		return err
	} else {
		return decoder.Decode(body)
	}
}

func boxedCheck(v interface{}, wants string) error {
	if reflect.ValueOf(v).Kind() != reflect.String {
		return &InvalidTypeError{"value type is not " + wants}
	}
	return nil
}

func decodeHook(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
	dataVal, ok := data.(map[string]interface{})
	if !ok || len(dataVal) != 1 {
		return data, nil
	}

	for k, v := range dataVal {
		switch typeTag(k) {
		case typeTagInt:
			if err := boxedCheck(v, "int"); err != nil {
				return nil, err
			}

			if i, err := strconv.Atoi(v.(string)); err != nil {
				return nil, &InvalidTypeError{"value is not an int"}
			} else {
				return i, nil
			}

		case typeTagLong:
			if err := boxedCheck(v, "long"); err != nil {
				return nil, err
			}

			if i, err := strconv.ParseInt(v.(string), 10, 64); err != nil {
				return nil, &InvalidTypeError{"value is not an long"}
			} else {
				return i, nil
			}

		case typeTagDouble:
			if err := boxedCheck(v, "double"); err != nil {
				return nil, err
			}

			if f, err := strconv.ParseFloat(v.(string), 64); err != nil {
				return nil, &InvalidTypeError{"value is not a double"}
			} else {
				return f, nil
			}

		case typeTagDate:
			if err := boxedCheck(v, "date"); err != nil {
				return nil, err
			}

			if t, err := time.Parse(dateFormat, v.(string)); err != nil {
				return nil, &InvalidTypeError{"value is not a date [" + err.Error() + "]"}
			} else {
				return t, nil
			}

		case typeTagTime:
			if err := boxedCheck(v, "time"); err != nil {
				return nil, err
			}

			if t, err := time.Parse(timeFormat, v.(string)); err != nil {
				return nil, &InvalidTypeError{"value is not a time [" + err.Error() + "]"}
			} else {
				return t, nil
			}

		case typeTagDoc:
			if err := boxedCheck(v, "doc"); err != nil {
				return nil, err
			}

			parts := strings.Split(v.(string), ":")
			if len(parts) != 2 {
				return nil, &InvalidTypeError{"value is not a doc ref"}
			}
			return DocumentReference{parts[0], parts[1]}, nil

		case typeTagMod:
			if err := boxedCheck(v, "mod"); err != nil {
				return nil, err
			}

			return Module{v.(string)}, nil

		case typeTagObject:
			return v, nil

		default:
			return data, nil
		}
	}

	return data, nil
}

func marshal(v interface{}) ([]byte, error) {
	if enc, err := encode(v, ""); err != nil {
		return nil, err
	} else {
		return json.Marshal(enc)
	}
}

func encode(v interface{}, hint string) (interface{}, error) {
	if t, ok := v.(time.Time); ok {
		return encodeTime(t, hint)
	}

	if m, ok := v.(map[string]interface{}); ok {
		return encodeMap(m)
	}

	if s, ok := v.([]interface{}); ok {
		return encodeSlice(s)
	}

	if d, ok := v.(DocumentReference); ok {
		return encodeDocRef(d)
	}

	if m, ok := v.(Module); ok {
		return encodeMod(m)
	}

	switch reflect.ValueOf(v).Kind() {
	case reflect.Ptr:
		return encode(reflect.Indirect(reflect.ValueOf(v)).Interface(), hint)
	case reflect.Struct:
		return encodeStruct(v)
	}

	return v, nil
}

func encodeTime(t time.Time, hint string) (interface{}, error) {
	out := make(map[typeTag]interface{})
	if hint == "date" {
		out[typeTagDate] = t.Format(dateFormat)
	} else {
		out[typeTagTime] = t.Format(timeFormat)
	}
	return out, nil
}

func encodeDocRef(d DocumentReference) (interface{}, error) {
	return map[typeTag]string{typeTagDoc: d.CollectionName + ":" + d.RefID}, nil
}

func encodeMod(m Module) (interface{}, error) {
	return map[typeTag]string{typeTagMod: m.Name}, nil
}

func encodeMap(m map[string]interface{}) (interface{}, error) {
	out := make(map[string]interface{})

	for k, v := range m {
		if enc, err := encode(v, ""); err != nil {
			return nil, err
		} else {
			out[k] = enc
		}
	}

	return out, nil
}

func encodeSlice(s []interface{}) (interface{}, error) {
	out := make([]interface{}, len(s))
	for i, v := range s {
		if enc, err := encode(v, ""); err != nil {
			return nil, err
		} else {
			out[i] = enc
		}
	}

	return out, nil
}

func encodeStruct(s interface{}) (interface{}, error) {
	out := make(map[string]interface{})

	elem := reflect.ValueOf(s)
	fields := reflect.TypeOf(s).NumField()

	for i := 0; i < fields; i++ {
		structField := elem.Type().Field(i)
		tag, found := structField.Tag.Lookup(fieldTag)
		if !found {
			continue
		}

		tags := strings.Split(tag, ",")

		typeHint := ""
		if len(tags) > 1 {
			typeHint = tags[1]
		}

		if enc, err := encode(elem.Field(i).Interface(), typeHint); err != nil {
			return nil, err
		} else {
			name := tags[0]
			if name == "" {
				name = structField.Name
			}
			out[name] = enc
		}
	}

	return out, nil
}
