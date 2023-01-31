package fauna

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

func Marshal(a interface{}) ([]byte, error) {
	val, mapErr := faunaMap(a)
	if mapErr != nil {
		return nil, mapErr
	}

	return json.Marshal(val)
}

func faunaMap(v interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{}

	val := reflect.Indirect(reflect.ValueOf(v))

	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		indexField := val.Type().Field(i)

		key := indexField.Name
		if faunaKey, hasFaunaKey := indexField.Tag.Lookup("fauna"); hasFaunaKey {
			key = faunaKey
		}

		fieldInterface := val.Field(i).Interface()

		faunaType, hasFaunaTypeTag := indexField.Tag.Lookup("faunaType")
		if hasFaunaTypeTag {
			switch faunaType {
			case TagDate, TagInt, TagLong, TagModule, TagObj, TagTime, TagDouble:
				// valid tags
			case "-":
				continue
			default:
				return nil, fmt.Errorf("unsupported fauna tag [%s] on struct field [%s]", faunaType, indexField.Name)
			}
		}

		setFaunaTag := func(val string) {
			// allow users to override
			if faunaType != "" {
				return
			}

			faunaType = val
		}

		switch f.Kind() {
		case reflect.Struct:
			if indexField.Type == reflect.TypeOf(time.Time{}) {
				ts := fieldInterface.(time.Time)
				if ts.Hour() == 0 && ts.Minute() == 0 && ts.Second() == 0 {
					setFaunaTag(TagDate)
				} else {
					setFaunaTag(TagTime)
				}
			} else {
				childMap, childErr := faunaMap(f.Interface())
				if childErr != nil {
					return nil, childErr
				}
				result[key] = tagged(childMap, faunaType)
			}
		case reflect.Int64:
			setFaunaTag(TagLong)
		case reflect.Int32, reflect.Int:
			setFaunaTag(TagInt)
		case reflect.Float32, reflect.Float64:
			setFaunaTag(TagDouble)
		default:
			setFaunaTag("")
		}

		// don't overwrite structs
		if _, exists := result[key]; !exists {
			result[key] = tagged(fieldInterface, faunaType)
		}

	}

	return result, nil
}

func tagged(v interface{}, faunaTag string) interface{} {
	if faunaTag != "" {
		return map[string]interface{}{faunaTag: v}
	}

	return v
}
