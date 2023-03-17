package fauna

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

const (
	fieldTag = "fauna"

	dateFormat    = "2006-01-02"
	timeEncFormat = "2006-01-02T15:04:05.999999-07:00"
	timeDecFormat = "2006-01-02T15:04:05.999999Z"

	maxInt  = 2147483647
	minInt  = -2147483648
	maxLong = 9223372036854775807
	minLong = -9223372036854775808
)

type typeTag string

const (
	typeTagInt    typeTag = "@int"
	typeTagLong   typeTag = "@long"
	typeTagDouble typeTag = "@double"
	typeTagDate   typeTag = "@date"
	typeTagTime   typeTag = "@time"
	typeTagDoc    typeTag = "@doc"
	typeTagRef    typeTag = "@ref"
	typeTagSet    typeTag = "@set"
	typeTagMod    typeTag = "@mod"
	typeTagObject typeTag = "@object"
)

func keyConflicts(key string) bool {
	switch typeTag(key) {
	case typeTagInt, typeTagLong, typeTagDouble,
		typeTagDate, typeTagTime,
		typeTagDoc, typeTagMod, typeTagObject:
		return true
	default:
		return false
	}
}

type Module struct {
	Name string
}

type Document struct {
	ID   string     `fauna:"id"`
	Coll *Module    `fauna:"coll"`
	TS   *time.Time `fauna:"ts"`
	Data map[string]interface{}
}

type NamedDocument struct {
	Name string     `fauna:"name"`
	Coll *Module    `fauna:"coll"`
	TS   *time.Time `fauna:"ts"`
	Data map[string]interface{}
}

type Ref struct {
	ID   string  `fauna:"id"`
	Coll *Module `fauna:"coll"`
}

type NamedRef struct {
	Name string  `fauna:"name"`
	Coll *Module `fauna:"coll"`
}

type SetCollection struct {
	Data  []interface{} `fauna:"data"`
	After string        `fauna:"after"`
}

func mapDecoder(into interface{}) (*mapstructure.Decoder, error) {
	return mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:              "fauna",
		Result:               into,
		IgnoreUntaggedFields: true,
		ErrorUnused:          false,
		ErrorUnset:           false,
		DecodeHook:           unmarshalDoc,
		Squash:               true,
	})
}

func unmarshal(body []byte, into interface{}) error {
	decBody, err := decode(body)
	if err != nil {
		return err
	}

	dec, err := mapDecoder(into)
	if err != nil {
		return err
	}

	return dec.Decode(decBody)
}

var (
	docType      = reflect.TypeOf(&Document{})
	namedDocType = reflect.TypeOf(&NamedDocument{})
)

func unmarshalDoc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f != docType && f != namedDocType {
		return data, nil
	}

	var docData map[string]interface{}
	if f == docType {
		doc := data.(*Document)
		docData = doc.Data
		docData["id"] = doc.ID
		docData["coll"] = doc.Coll
		docData["ts"] = doc.TS
	}

	if f == namedDocType {
		doc := data.(*NamedDocument)
		docData = doc.Data
		docData["name"] = doc.Name
		docData["coll"] = doc.Coll
		docData["ts"] = doc.TS
	}

	result := reflect.New(t).Interface()
	dec, err := mapDecoder(result)
	if err != nil {
		return nil, err
	}

	if err := dec.Decode(docData); err != nil {
		return nil, err
	}

	return result, nil
}

func decode(bodyBytes []byte) (interface{}, error) {
	var body interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return nil, err
	}

	return convert(false, body)
}

func convert(escaped bool, body interface{}) (interface{}, error) {
	switch b := body.(type) {
	case map[string]interface{}:
		if escaped {
			return convertMap(b)
		} else {
			return unboxType(b)
		}

	case []interface{}:
		return convertSlice(b)

	default:
		return body, nil
	}
}

func convertMap(body map[string]interface{}) (map[string]interface{}, error) {
	retBody := map[string]interface{}{}
	for k, vRaw := range body {
		if v, err := convert(false, vRaw); err != nil {
			return nil, err
		} else {
			retBody[k] = v
		}
	}
	return retBody, nil
}

func convertSlice(body []interface{}) ([]interface{}, error) {
	for i, vRaw := range body {
		if v, err := convert(false, vRaw); err != nil {
			return nil, err
		} else {
			body[i] = v
		}
	}
	return body, nil
}

func unboxType(body map[string]interface{}) (interface{}, error) {
	if len(body) == 1 {
		for boxedK, v := range body {
			switch typeTag(boxedK) {
			case typeTagInt, typeTagLong:
				return unboxInt(v.(string))
			case typeTagDouble:
				return unboxDouble(v.(string))
			case typeTagDate:
				return unboxDate(v.(string))
			case typeTagTime:
				return unboxTime(v.(string))
			case typeTagMod:
				return unboxMod(v.(string))
			case typeTagRef:
				return unboxRef(v.(map[string]interface{}))
			case typeTagSet:
				return unboxSet(v.(map[string]interface{}))
			case typeTagDoc:
				return unboxDoc(v.(map[string]interface{}))
			case typeTagObject:
				return convertMap(v.(map[string]interface{}))
			}
		}
	}

	return convertMap(body)
}

func unboxMod(v string) (*Module, error) {
	m := Module{v}
	return &m, nil
}

func getColl(v map[string]interface{}) (*Module, error) {
	if coll, ok := v["coll"]; ok {
		modI, err := convert(false, coll)
		if err != nil {
			return nil, err
		}

		if mod, ok := modI.(*Module); ok {
			return mod, nil
		}
	}
	return nil, nil
}

func getIDorName(v map[string]interface{}) (id string, name string) {
	if idRaw, ok := v["id"]; ok {
		if id, ok := idRaw.(string); ok {
			return id, ""
		}
	}

	if nameRaw, ok := v["name"]; ok {
		if name, ok := nameRaw.(string); ok {
			return "", name
		}
	}

	return
}

func unboxRef(v map[string]interface{}) (interface{}, error) {
	mod, err := getColl(v)
	if err != nil {
		return nil, err
	}

	if mod != nil {
		id, name := getIDorName(v)
		if id != "" {
			return &Ref{id, mod}, nil
		}

		if name != "" {
			return &NamedRef{name, mod}, nil
		}
	}

	return nil, fmt.Errorf("invalid ref %v", v)
}

func unboxDoc(v map[string]interface{}) (interface{}, error) {
	mod, err := getColl(v)
	if err != nil {
		return nil, err
	}

	var ts *time.Time
	if tsRaw, ok := v["ts"]; ok {
		if tsI, err := convert(false, tsRaw); err != nil {
			return nil, err
		} else {
			if unboxedTS, ok := tsI.(*time.Time); ok {
				ts = unboxedTS
			}
		}
	}

	id, name := getIDorName(v)

	if mod != nil && ts != nil && (id != "" || name != "") {
		delete(v, "id")
		delete(v, "coll")
		delete(v, "ts")

		if id == "" {
			delete(v, "name")
		}

		data, err := convertMap(v)
		if err != nil {
			return nil, err
		}

		if id != "" {
			return &Document{ID: id, Coll: mod, TS: ts, Data: data}, nil
		}

		if name != "" {
			return &NamedDocument{Name: name, Coll: mod, TS: ts, Data: data}, nil
		}
	}

	return nil, fmt.Errorf("invalid doc %v", v)
}

func unboxSet(v map[string]interface{}) (interface{}, error) {
	if dataI, ok := v["data"]; ok {
		if dataRaw, ok := dataI.([]interface{}); ok {
			data, err := convertSlice(dataRaw)
			if err != nil {
				return nil, err
			}

			setC := SetCollection{Data: data}
			if afterRaw, ok := v["after"]; ok {
				if after, ok := afterRaw.(string); ok {
					setC.After = after
				}
			}

			return &setC, nil
		}
	}

	return nil, fmt.Errorf("invalid set %v", v)
}

func unboxTime(v string) (*time.Time, error) {
	if t, err := time.Parse(timeDecFormat, v); err != nil {
		return nil, err
	} else {
		return &t, nil
	}
}

func unboxDate(v string) (*time.Time, error) {
	if t, err := time.Parse(dateFormat, v); err != nil {
		return nil, err
	} else {
		return &t, nil
	}
}

func unboxInt(v string) (interface{}, error) {
	if i, err := strconv.ParseInt(v, 10, 64); err != nil {
		return nil, err
	} else {
		return i, nil
	}
}

func unboxDouble(v string) (interface{}, error) {
	if i, err := strconv.ParseFloat(v, 64); err != nil {
		return nil, err
	} else {
		return i, nil
	}
}

func marshal(v interface{}) ([]byte, error) {
	if enc, err := encode(v, ""); err != nil {
		return nil, err
	} else {
		return json.Marshal(enc)
	}
}

func encode(v interface{}, hint string) (interface{}, error) {
	switch vt := v.(type) {
	case Module:
		return encodeMod(vt)

	case Ref:
		return encodeFaunaStruct(typeTagRef, vt)

	case NamedRef:
		return encodeFaunaStruct(typeTagRef, vt)

	case SetCollection:
		return encodeFaunaStruct(typeTagSet, vt)

	case time.Time:
		return encodeTime(vt, hint)

	case fqlRequest:
		out := map[string]interface{}{"query": vt.Query}
		if len(vt.Arguments) > 0 {
			if args, err := encodeMap(reflect.ValueOf(vt.Arguments)); err != nil {
				return nil, err
			} else {
				out["arguments"] = args
			}
		}
		return out, nil
	}

	switch value := reflect.ValueOf(v); value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i := value.Int(); i < minLong {
			return nil, fmt.Errorf("numeric value is outside Fauna's type constraints")
		} else {
			return encodeInt(i)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if i := value.Uint(); i > maxLong {
			return nil, fmt.Errorf("numeric value is outside Fauna's type constraints")
		} else {
			return encodeInt(int64(i))
		}

	case reflect.Float32, reflect.Float64:
		return map[typeTag]interface{}{typeTagDouble: strconv.FormatFloat(value.Float(), 'f', -1, 64)}, nil

	case reflect.Ptr:
		if value.IsNil() {
			return nil, nil
		}
		return encode(reflect.Indirect(value).Interface(), hint)

	case reflect.Struct:
		return encodeStruct(v)

	case reflect.Map:
		return encodeMap(value)

	case reflect.Slice:
		return encodeSlice(value)
	}

	return v, nil
}

func encodeInt(i int64) (interface{}, error) {
	tag := typeTagLong
	if i <= maxInt && i >= minInt {
		tag = typeTagInt
	}
	return map[typeTag]interface{}{tag: strconv.FormatInt(i, 10)}, nil
}

func encodeTime(t time.Time, hint string) (interface{}, error) {
	out := make(map[typeTag]interface{})
	if hint == "date" {
		out[typeTagDate] = t.Format(dateFormat)
	} else {
		out[typeTagTime] = t.Format(timeEncFormat)
	}
	return out, nil
}

func encodeMod(m Module) (interface{}, error) {
	return map[typeTag]string{typeTagMod: m.Name}, nil
}

func encodeFaunaStruct(tag typeTag, s interface{}) (interface{}, error) {
	if doc, err := encodeStruct(s); err != nil {
		return nil, err
	} else {
		return map[typeTag]interface{}{tag: doc}, nil
	}
}

func encodeMap(mv reflect.Value) (interface{}, error) {
	hasConflictingKey := false
	out := make(map[string]interface{})

	mi := mv.MapRange()
	for i := 0; mi.Next(); i++ {
		if mi.Key().Kind() != reflect.String {
			return mv.Interface(), nil
		}

		if enc, err := encode(mi.Value().Interface(), ""); err != nil {
			return nil, err
		} else {

			key := mi.Key().String()
			if keyConflicts(key) {
				hasConflictingKey = true
			}
			out[key] = enc
		}
	}

	if hasConflictingKey {
		return map[typeTag]interface{}{typeTagObject: out}, nil
	} else {
		return out, nil
	}
}

func encodeSlice(sv reflect.Value) (interface{}, error) {
	sLen := sv.Len()
	out := make([]interface{}, sLen)
	for i := 0; i < sLen; i++ {
		if enc, err := encode(sv.Index(i).Interface(), ""); err != nil {
			return nil, err
		} else {
			out[i] = enc
		}
	}

	return out, nil
}

func encodeStruct(s interface{}) (interface{}, error) {
	hasConflictingKey := false
	isDoc := false
	out := make(map[string]interface{})

	elem := reflect.ValueOf(s)
	fields := reflect.TypeOf(s).NumField()

	for i := 0; i < fields; i++ {
		structField := elem.Type().Field(i)

		if structField.Anonymous && structField.Name == "Document" {
			doc := elem.Field(i).Interface().(Document)
			out["id"] = doc.ID

			if coll, err := encode(doc.Coll, ""); err != nil {
				return nil, err
			} else {
				out["coll"] = coll
			}

			if ts, err := encode(doc.TS, "time"); err != nil {
				return nil, err
			} else {
				out["ts"] = ts
			}

			isDoc = true
			continue
		}

		if structField.Anonymous && structField.Name == "NamedDocument" {
			doc := elem.Field(i).Interface().(NamedDocument)
			out["name"] = doc.Name

			if coll, err := encode(doc.Coll, ""); err != nil {
				return nil, err
			} else {
				out["coll"] = coll
			}

			if ts, err := encode(doc.TS, "time"); err != nil {
				return nil, err
			} else {
				out["ts"] = ts
			}

			isDoc = true
			continue
		}

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

			if keyConflicts(name) {
				hasConflictingKey = true
			}
			out[name] = enc
		}
	}

	if isDoc {
		return map[typeTag]interface{}{typeTagDoc: out}, nil
	}

	if hasConflictingKey {
		return map[typeTag]interface{}{typeTagObject: out}, nil
	}

	return out, nil
}
