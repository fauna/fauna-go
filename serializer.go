package fauna

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
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

type DocumentReference struct {
	CollectionName string
	RefID          string
}

type Module struct {
	Name string
}

func typeErr(expected string, data interface{}) error {
	return fmt.Errorf("%s is not a %s", reflect.ValueOf(data).Kind(), expected)
}

func unmarshal(body []byte, into interface{}) error {
	intoVal := reflect.ValueOf(into)
	if intoVal.Kind() != reflect.Ptr {
		return fmt.Errorf("result must be a pointer got %s", intoVal.Kind())
	}

	intoVal = intoVal.Elem()
	if !intoVal.CanAddr() {
		return errors.New("result must be addressable (a pointer)")
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var bodyMap interface{}
	if err := dec.Decode(&bodyMap); err != nil {
		return err
	}

	return decode(false, bodyMap, intoVal)
}

func decode(escaped bool, body interface{}, intoVal reflect.Value) error {
	if body == nil {
		return nil
	}

	bodyVal := reflect.ValueOf(body)

	if !bodyVal.IsValid() {
		// If the input value is invalid, then we just set the value
		// to be the zero value.
		intoVal.Set(reflect.Zero(intoVal.Type()))
		return nil
	}

	switch intoVal.Interface().(type) {
	case time.Time:
		return decodeTime(body, intoVal)
	case DocumentReference:
		return decodeDoc(body, intoVal)
	case Module:
		return decodeMod(body, intoVal)
	}

	switch intoVal.Kind() {
	case reflect.Bool:
		return decodeBool(body, intoVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:
		return decodeInt(body, intoVal)
	case reflect.Float32, reflect.Float64:
		return decodeFloat(body, intoVal)
	case reflect.String:
		return decodeString(body, intoVal)
	case reflect.Slice:
		return decodeSlice(body, intoVal)
	case reflect.Array:
		return decodeArray(body, intoVal)
	case reflect.Struct:
		return decodeStruct(escaped, body, intoVal)
	case reflect.Map:
		return decodeMap(escaped, body, intoVal)
	case reflect.Ptr:
		return decodePtr(escaped, body, intoVal)
	}

	return nil
}

func unbox(tag typeTag, body interface{}) (string, bool) {
	if m, ok := body.(map[string]interface{}); ok && len(m) == 1 {
		if v, ok := m[string(tag)]; ok {
			if out, ok := v.(string); ok {
				return out, true
			}
		}
	}
	return "", false
}

func decodeDoc(body interface{}, into reflect.Value) error {
	if val, ok := unbox(typeTagDoc, body); ok {
		parts := strings.Split(val, ":")
		if len(parts) == 2 {
			into.Set(reflect.ValueOf(DocumentReference{parts[0], parts[1]}))
			return nil
		}
	}
	return typeErr("fauna.DocumentReference", body)
}

func decodeMod(body interface{}, into reflect.Value) error {
	if val, ok := unbox(typeTagMod, body); ok {
		into.Set(reflect.ValueOf(Module{val}))
		return nil
	}
	return typeErr("fauna.Module", body)
}

func decodeTime(body interface{}, into reflect.Value) error {
	if val, ok := unbox(typeTagDate, body); ok {
		if t, err := time.Parse(dateFormat, val); err != nil {
			return err
		} else {
			into.Set(reflect.ValueOf(t))
			return nil
		}
	}

	if val, ok := unbox(typeTagTime, body); ok {
		if t, err := time.Parse(timeDecFormat, val); err != nil {
			return err
		} else {
			into.Set(reflect.ValueOf(t))
			return nil
		}
	}

	return typeErr("time.Time", body)
}

func decodeBool(body interface{}, into reflect.Value) error {
	if b, ok := body.(bool); !ok {
		return typeErr("bool", body)
	} else {
		into.SetBool(b)
		return nil
	}
}

func decodeInt(body interface{}, into reflect.Value) error {
	bodyVal := reflect.ValueOf(body)
	bodyKind := bodyVal.Kind()
	switch {
	case bodyKind >= reflect.Int && bodyKind <= reflect.Int64:
		into.SetInt(bodyVal.Int())
		return nil
	case bodyKind >= reflect.Uint && bodyKind <= reflect.Uint64:
		into.SetInt(int64(bodyVal.Uint()))
		return nil
	case bodyKind >= reflect.Float32 && bodyKind <= reflect.Float64:
		into.SetInt(int64(bodyVal.Float()))
		return nil
	case bodyKind == reflect.String:
		str := bodyVal.String()
		if str == "" {
			str = "0"
		}

		if i, err := strconv.ParseInt(str, 10, into.Type().Bits()); err != nil {
			return err
		} else {
			into.SetInt(i)
			return nil
		}
	}

	val := ""
	ok := false
	if val, ok = unbox(typeTagInt, body); !ok {
		if val, ok = unbox(typeTagLong, body); !ok {
			return typeErr("int", body)
		}
	}

	if i, err := strconv.ParseInt(val, 10, into.Type().Bits()); err != nil {
		return err
	} else {
		into.SetInt(i)
		return nil
	}
}

func decodeFloat(body interface{}, into reflect.Value) error {
	bodyVal := reflect.ValueOf(body)
	bodyKind := bodyVal.Kind()
	switch {
	case bodyKind >= reflect.Int && bodyKind <= reflect.Int64:
		into.SetFloat(float64(bodyVal.Int()))
		return nil
	case bodyKind >= reflect.Uint && bodyKind <= reflect.Uint64:
		into.SetFloat(float64(bodyVal.Uint()))
		return nil
	case bodyKind >= reflect.Float32 && bodyKind <= reflect.Float64:
		into.SetFloat(bodyVal.Float())
		return nil
	case bodyKind == reflect.String:
		str := bodyVal.String()
		if str == "" {
			str = "0"
		}

		if f, err := strconv.ParseFloat(str, into.Type().Bits()); err != nil {
			return err
		} else {
			into.SetFloat(f)
			return nil
		}
	}

	if val, ok := unbox(typeTagDouble, body); !ok {
		return typeErr("float", body)

	} else {
		if f, err := strconv.ParseFloat(val, into.Type().Bits()); err != nil {
			return err
		} else {
			into.SetFloat(f)
			return nil
		}
	}
}

func decodeString(body interface{}, into reflect.Value) error {
	if s, ok := body.(string); !ok {
		into.SetString(s)
		return nil
	}

	bodyVal := reflect.ValueOf(body)
	if bodyVal.Kind() == reflect.String {
		into.SetString(bodyVal.String())
		return nil
	}

	return typeErr("string", body)
}

func decodeSlice(body interface{}, into reflect.Value) error {
	bodyVal := reflect.ValueOf(body)
	bodyKind := bodyVal.Kind()
	if bodyKind != reflect.Array && bodyKind != reflect.Slice {
		return fmt.Errorf("%s is not an array", bodyKind)
	}

	intoType := into.Type()
	intoElemType := intoType.Elem()

	intoSlice := into
	if intoSlice.IsNil() {
		// Make a new slice to hold our result, same size as the original data.
		intoSlice = reflect.MakeSlice(intoType, bodyVal.Len(), bodyVal.Len())
	}

	for i := 0; i < bodyVal.Len(); i++ {
		currentData := bodyVal.Index(i)
		for intoSlice.Len() <= i {
			intoSlice = reflect.Append(intoSlice, reflect.Zero(intoElemType))
		}
		currentField := intoSlice.Index(i)
		if err := decode(false, currentData, currentField); err != nil {
			return err
		}
	}

	// Finally, set the value to the slice we built up
	into.Set(intoSlice)

	return nil
}

func decodeArray(body interface{}, into reflect.Value) error {
	bodyVal := reflect.ValueOf(body)
	bodyKind := bodyVal.Kind()
	intoType := into.Type()
	intoElemType := intoType.Elem()
	arrayType := reflect.ArrayOf(intoType.Len(), intoElemType)

	valArray := into

	if valArray.Interface() == reflect.Zero(valArray.Type()).Interface() {
		// Check input type
		if bodyKind != reflect.Array && bodyKind != reflect.Slice {
			return fmt.Errorf(
				"source data must be an array or slice, got %s", bodyKind)

		}
		if into.Len() > arrayType.Len() {
			return fmt.Errorf(
				"expected source data to have length less or equal to %d, got %d", arrayType.Len(), bodyVal.Len())

		}

		// Make a new array to hold our result, same size as the original data.
		valArray = reflect.New(arrayType).Elem()
	}

	for i := 0; i < bodyVal.Len(); i++ {
		currentData := bodyVal.Index(i)
		currentField := valArray.Index(i)

		if err := decode(false, currentData.Interface(), currentField); err != nil {
			return err
		}
	}

	// Finally, set the value to the array we built up
	into.Set(valArray)

	return nil
}

func unboxObject(body interface{}) (interface{}, bool) {
	bodyVal := reflect.ValueOf(body)
	key := bodyVal.MapKeys()[0]
	if key.Kind() == reflect.String && typeTag(key.String()) == typeTagObject {
		return bodyVal.MapIndex(key).Interface(), true
	}

	return nil, false
}

func decodeStruct(escaped bool, body interface{}, into reflect.Value) error {
	bodyVal := reflect.ValueOf(body)

	if bodyVal.Kind() != reflect.Map {
		return typeErr("object", body)
	}

	// If the input data is empty, then we just match what the input data is.
	if bodyVal.Len() == 0 {
		if bodyVal.IsNil() {
			if !into.IsNil() {
				into.Set(bodyVal)
			}
		}

		return nil
	}

	if !escaped {
		if unboxed, ok := unboxObject(body); ok {
			return decode(true, unboxed, into)
		}
	}

	structType := into.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := into.Field(i)
		if fieldVal.Kind() == reflect.Ptr && fieldVal.Elem().Kind() == reflect.Struct {
			// Handle embedded struct pointers as embedded structs.
			fieldVal = fieldVal.Elem()
		}

		fieldName := field.Name
		if n := strings.Split(field.Tag.Get(fieldTag), ",")[0]; n != "" {
			fieldName = n
		}

		rawMapKey := reflect.ValueOf(fieldName)
		rawMapVal := bodyVal.MapIndex(rawMapKey)
		if !rawMapVal.IsValid() {
			continue
		}

		if !fieldVal.IsValid() {
			// This should never happen
			panic("field is not valid")
		}

		if !fieldVal.CanSet() {
			continue
		}

		if err := decode(false, rawMapVal.Interface(), fieldVal); err != nil {
			return err
		}
	}

	return nil
}

func decodeMap(escaped bool, body interface{}, into reflect.Value) error {
	bodyVal := reflect.ValueOf(body)
	if bodyVal.Kind() != reflect.Map {
		return typeErr("map", body)
	}

	// If the input data is empty, then we just match what the input data is.
	if bodyVal.Len() == 0 {
		if bodyVal.IsNil() {
			if !into.IsNil() {
				into.Set(bodyVal)
			}
		} else {
			// Set to empty allocated value
			into.Set(bodyVal)
		}

		return nil
	}

	if !escaped {
		if unboxed, ok := unboxObject(body); ok {
			return decode(true, unboxed, into)
		}
	}

	intoType := into.Type()
	intoKeyType := intoType.Key()
	intoElemType := intoType.Elem()

	// By default we overwrite keys in the current map
	intoMap := into

	// If the map is nil or we're purposely zeroing fields, make a new map
	if intoMap.IsNil() {
		// Make a new map to hold our result
		mapType := reflect.MapOf(intoKeyType, intoElemType)
		intoMap = reflect.MakeMap(mapType)
	}

	for _, k := range bodyVal.MapKeys() {
		// First decode the key into the proper type
		currentKey := reflect.Indirect(reflect.New(intoKeyType))
		if err := decode(false, k.Interface(), currentKey); err != nil {
			return err
		}

		// Next decode the data into the proper type
		v := bodyVal.MapIndex(k)
		currentVal := reflect.Indirect(reflect.New(intoElemType))
		if err := decode(false, v.Interface(), currentVal); err != nil {
			return err
		}

		intoMap.SetMapIndex(currentKey, currentVal)
	}

	// Set the built up map to the value
	into.Set(intoMap)

	return nil
}

func decodePtr(escaped bool, body interface{}, into reflect.Value) error {
	isNil := body == nil
	if !isNil {
		switch v := reflect.Indirect(reflect.ValueOf(body)); v.Kind() {
		case reflect.Chan,
			reflect.Func,
			reflect.Interface,
			reflect.Map,
			reflect.Ptr,
			reflect.Slice:
			isNil = v.IsNil()
		}
	}

	if isNil {
		if !into.IsNil() && into.CanSet() {
			nilValue := reflect.New(into.Type()).Elem()
			into.Set(nilValue)
		}

		return nil
	}

	// Create an element of the concrete (non pointer) type and decode
	// into that. Then set the value of the pointer to this type.
	valType := into.Type()
	valElemType := valType.Elem()
	if into.CanSet() {
		realVal := into
		if realVal.IsNil() {
			realVal = reflect.New(valElemType)
		}

		if err := decode(escaped, body, reflect.Indirect(realVal)); err != nil {
			return err
		}

		into.Set(realVal)
	} else {
		if err := decode(escaped, body, reflect.Indirect(into)); err != nil {
			return err
		}
	}

	return nil
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
	case time.Time:
		return encodeTime(vt, hint)

	case DocumentReference:
		return encodeDocRef(vt)

	case Module:
		return encodeMod(vt)

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

func encodeDocRef(d DocumentReference) (interface{}, error) {
	return map[typeTag]string{typeTagDoc: d.CollectionName + ":" + d.RefID}, nil
}

func encodeMod(m Module) (interface{}, error) {
	return map[typeTag]string{typeTagMod: m.Name}, nil
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

			if keyConflicts(name) {
				hasConflictingKey = true
			}
			out[name] = enc
		}
	}

	if hasConflictingKey {
		return map[typeTag]interface{}{typeTagObject: out}, nil
	} else {
		return out, nil
	}
}
