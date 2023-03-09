package fauna

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type SubBusinessObj struct {
	StringField       string                 `fauna:"string_field"`
	BoolField         bool                   `fauna:"bool_field"`
	MapField          map[string]interface{} `fauna:"map_field"`
	SingleKeyMapField map[string]interface{} `fauna:"single_key_map_field"`
	SliceField        []interface{}          `fauna:"slice_field"`
	TupleField        []interface{}          `fauna:"tuple_field"`
	IntField          int                    `fauna:"int_field"`
	DoubleField       float64                `fauna:"double_field"`

	IgnoredField2 string
}

type BusinessObj struct {
	IntField       int                `fauna:"int_field"`
	LongField      int64              `fauna:"long_field"`
	DoubleField    float64            `fauna:"double_field"`
	DocRefField    DocumentReference  `fauna:"doc_ref_field"`
	ModField       Module             `fauna:"mod_field"`
	PtrDocRefField *DocumentReference `fauna:"ptr_doc_ref_field"`
	PtrModField    *Module            `fauna:"ptr_mod_field"`
	ObjField       SubBusinessObj     `fauna:"obj_field"`

	IgnoredField string
}

func marshalAndCheck(t *testing.T, obj interface{}) []byte {
	if bs, err := marshal(obj); err != nil {
		t.Fatalf("failed to marshal: %s", err)
		return nil
	} else {
		return bs
	}
}

func unmarshalAndCheck(t *testing.T, bs []byte, obj interface{}) {
	if err := unmarshal(bs, obj); err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
}

func TestRoundtrip(t *testing.T) {
	var doc = `{
    "int_field": {"@int":"1234"},
    "long_field": {"@long":"123456"},
    "double_field": {"@double":"123.456"},
    "doc_ref_field": {"@doc":"Foo:123"},
    "mod_field": {"@mod":"Foo"},
    "ptr_doc_ref_field": {"@doc":"PtrFoo:123"},
    "ptr_mod_field": {"@mod":"PtrFoo"},
    "obj_field": {"@object": {
      "string_field": "foobarbaz",
      "bool_field": true,
      "map_field": {"foo":"bar","baz":"buz"},
      "single_key_map_field": {"foo":"bar"},
      "slice_field": [1,2,3,4],
      "tuple_field": ["one",2,3.0],
      "int_field": 1234,
      "double_field": 1234.567
    }}
  }`

	obj := &BusinessObj{}
	unmarshalAndCheck(t, []byte(doc), obj)

	bs := marshalAndCheck(t, obj)

	obj2 := &BusinessObj{}
	unmarshalAndCheck(t, bs, obj2)

	assert.Equal(t, obj, obj2)
}

func roundTripCheck(t *testing.T, test interface{}, expected string) {
	bs := marshalAndCheck(t, test)
	if assert.JSONEq(t, expected, string(bs)) {
		decoded := reflect.New(reflect.TypeOf(test)).Interface()
		unmarshalAndCheck(t, bs, decoded)
		dec := reflect.Indirect(reflect.ValueOf(decoded)).Interface()
		assert.EqualValues(t, test, dec)
	}
}

func TestEncodingPrimitives(t *testing.T) {
	t.Run("encode string", func(t *testing.T) {
		roundTripCheck(t, "foo", `"foo"`)
	})

	t.Run("encode bools", func(t *testing.T) {
		roundTripCheck(t, true, `true`)
		roundTripCheck(t, false, `false`)
	})

	t.Run("encode ints", func(t *testing.T) {
		roundTripCheck(t, int(10), `{"@int":"10"}`)
		roundTripCheck(t, int8(10), `{"@int":"10"}`)
		roundTripCheck(t, int16(10), `{"@int":"10"}`)
		roundTripCheck(t, int32(10), `{"@int":"10"}`)

		roundTripCheck(t, int(-10), `{"@int":"-10"}`)
		roundTripCheck(t, int8(-10), `{"@int":"-10"}`)
		roundTripCheck(t, int16(-10), `{"@int":"-10"}`)
		roundTripCheck(t, int32(-10), `{"@int":"-10"}`)

		roundTripCheck(t, 2147483647, `{"@int":"2147483647"}`)
		roundTripCheck(t, -2147483648, `{"@int":"-2147483648"}`)
	})

	t.Run("encode longs", func(t *testing.T) {
		roundTripCheck(t, 2147483648, `{"@long":"2147483648"}`)
		roundTripCheck(t, -2147483649, `{"@long":"-2147483649"}`)

		roundTripCheck(t, 9223372036854775807, `{"@long":"9223372036854775807"}`)
		roundTripCheck(t, -9223372036854775808, `{"@long":"-9223372036854775808"}`)
	})

	t.Run("fail on numbers that are too big", func(t *testing.T) {
		tooLarge := uint(9223372036854775808)
		_, tooLargeErr := marshal(tooLarge)
		assert.Error(t, tooLargeErr)
	})

	t.Run("encode floats", func(t *testing.T) {
		roundTripCheck(t, 100.0, `{"@double":"100"}`)
		roundTripCheck(t, -100.1, `{"@double":"-100.1"}`)
		roundTripCheck(t, 9.999999999999, `{"@double":"9.999999999999"}`)
	})

	t.Run("encode nil", func(t *testing.T) {
		bs := marshalAndCheck(t, nil)
		if assert.JSONEq(t, `null`, string(bs)) {
			var decoded *string
			unmarshalAndCheck(t, bs, &decoded)
			assert.Nil(t, decoded)
		}
	})
}

func TestEncodingTime(t *testing.T) {
	t.Run("encodes time as @time", func(t *testing.T) {
		if tz, err := time.LoadLocation("America/Los_Angeles"); assert.NoError(t, err) {
			bs := marshalAndCheck(t, time.Date(2023, 02, 28, 10, 10, 10, 10000, tz))
			if assert.JSONEq(t, `{"@time":"2023-02-28T10:10:10.00001-08:00"}`, string(bs)) {
				var decoded time.Time
				bs := []byte(`{"@time":"2023-02-28T18:10:10.000010Z"}`)
				unmarshalAndCheck(t, bs, &decoded)
				assert.Equal(t, time.Date(2023, 02, 28, 18, 10, 10, 10000, time.UTC), decoded)
			}
		}
	})

	t.Run("encodes time as @date when hinted", func(t *testing.T) {
		obj := struct {
			D time.Time `fauna:"d_field,date"`
		}{
			D: time.Date(2023, 02, 28, 0, 0, 0, 0, time.UTC),
		}
		roundTripCheck(t, obj, `{"d_field":{"@date":"2023-02-28"}}`)
	})
}

func TestEncodingStructs(t *testing.T) {
	t.Run("encodes using struct field names", func(t *testing.T) {
		obj := struct {
			Field string `fauna:""`
		}{"foo"}
		roundTripCheck(t, obj, `{"Field":"foo"}`)
	})

	t.Run("encodes using configured field names", func(t *testing.T) {
		obj := struct {
			Field string `fauna:"field_name"`
		}{"foo"}
		roundTripCheck(t, obj, `{"field_name":"foo"}`)
	})

	t.Run("encodes hinted types without configured name", func(t *testing.T) {
		obj := struct {
			Field time.Time `fauna:",date"`
		}{
			Field: time.Date(2023, 02, 28, 0, 0, 0, 0, time.UTC),
		}
		roundTripCheck(t, obj, `{"Field":{"@date":"2023-02-28"}}`)
	})

	t.Run("encodes hinted types with configured name", func(t *testing.T) {
		obj := struct {
			Field time.Time `fauna:"field_name,date"`
		}{
			Field: time.Date(2023, 02, 28, 0, 0, 0, 0, time.UTC),
		}
		roundTripCheck(t, obj, `{"field_name":{"@date":"2023-02-28"}}`)
	})

	t.Run("ignores untagged fields", func(t *testing.T) {
		obj := struct {
			Field        string `fauna:""`
			IgnoredField string
		}{
			Field:        "foo",
			IgnoredField: "",
		}
		roundTripCheck(t, obj, `{"Field":"foo"}`)
	})

	t.Run("encodes nested fields", func(t *testing.T) {
		var obj struct {
			GrandParent struct {
				Parent struct {
					Child   string `fauna:""`
					Sibling string `fauna:""`
				} `fauna:""`
			} `fauna:""`
		}
		obj.GrandParent.Parent.Child = "foo"
		obj.GrandParent.Parent.Sibling = "bar"

		roundTripCheck(t, obj, `{"GrandParent":{"Parent":{"Child":"foo","Sibling":"bar"}}}`)
	})
}

func TestEncodingPointers(t *testing.T) {
	type checkStruct struct {
		Field string `fauna:""`
	}

	var obj struct {
		NilPtrField *checkStruct `fauna:""`
		PtrField    *checkStruct `fauna:""`
		Field       checkStruct  `fauna:""`
	}
	obj.NilPtrField = nil
	obj.PtrField = &checkStruct{"foo"}
	obj.Field = checkStruct{"bar"}
	roundTripCheck(t, obj, `{"NilPtrField":null,"PtrField":{"Field":"foo"},"Field":{"Field":"bar"}}`)

}

func TestEncodingObject(t *testing.T) {
	t.Run("object has @object key", func(t *testing.T) {
		test := map[string]int{"@object": 10}
		expected := `{"@object":{"@object":{"@int":"10"}}}`
		bs := marshalAndCheck(t, test)

		assert.JSONEq(t, expected, string(bs))
	})

	t.Run("object has inner conflicting @int key", func(t *testing.T) {
		roundTripCheck(
			t,
			map[string]map[string]string{"@object": {"@int": "bar"}},
			`{"@object":{"@object":{"@object":{"@int":"bar"}}}}`,
		)
	})

	t.Run("object has inner conflicting @object key", func(t *testing.T) {
		roundTripCheck(
			t,
			map[string]map[string]string{"@object": {"@object": "bar"}},
			`{"@object":{"@object":{"@object":{"@object":"bar"}}}}`,
		)
	})

	t.Run("object has multiple conflicting type keys", func(t *testing.T) {
		roundTripCheck(
			t,
			map[string]string{"@int": "foo", "@double": "bar"},
			`{"@object":{"@int":"foo","@double":"bar"}}`,
		)
	})

	t.Run("object has mixed keys with a conflict", func(t *testing.T) {
		roundTripCheck(
			t,
			map[string]string{"@int": "foo", "bar": "buz"},
			`{"@object":{"@int":"foo","bar":"buz"}}`,
		)
	})

	t.Run("object has nested conflicting keys", func(t *testing.T) {
		roundTripCheck(
			t,
			map[string]map[string]map[string]map[string]int{"@int": {"@date": {"@time": {"@long": 10}}}},
			`{"@object":{"@int":{"@object":{"@date":{"@object":{"@time":{"@object":{"@long":{"@int":"10"}}}}}}}}}`,
		)
	})

	t.Run("object has non-conflicting keys", func(t *testing.T) {
		roundTripCheck(
			t,
			map[string]int{"@foo": 10},
			`{"@foo":{"@int":"10"}}`,
		)
	})
}
