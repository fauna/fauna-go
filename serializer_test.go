package fauna

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type SubBusinessObj struct {
	StringField       string         `fauna:"string_field"`
	BoolField         bool           `fauna:"bool_field"`
	MapField          map[string]any `fauna:"map_field"`
	SingleKeyMapField map[string]any `fauna:"single_key_map_field"`
	SliceField        []any          `fauna:"slice_field"`
	TupleField        []any          `fauna:"tuple_field"`
	IntField          int            `fauna:"int_field"`
	DoubleField       float64        `fauna:"double_field"`

	IgnoredField2 string
}

type DocBusinessObj struct {
	Document
	ExtraField string `fauna:"extra_field"`
}

type NamedDocBusinessObj struct {
	NamedDocument
	ExtraField string `fauna:"extra_field"`
}

type BusinessObj struct {
	IntField      int                 `fauna:"int_field"`
	LongField     int64               `fauna:"long_field"`
	DoubleField   float64             `fauna:"double_field"`
	PtrModField   *Module             `fauna:"ptr_mod_field"`
	ModField      Module              `fauna:"mod_field"`
	PtrRefField   *Ref                `fauna:"ptr_ref_field"`
	RefField      Ref                 `fauna:"ref_field"`
	NamedRefField NamedRef            `fauna:"named_ref_field"`
	SetField      Page                `fauna:"set_field"`
	ObjField      SubBusinessObj      `fauna:"obj_field"`
	DocField      DocBusinessObj      `fauna:"doc_field"`
	NamedDocField NamedDocBusinessObj `fauna:"named_doc_field"`

	IgnoredField string
}

func marshalAndCheck(t *testing.T, obj any) []byte {
	if bs, err := marshal(obj); err != nil {
		t.Fatalf("failed to marshal: %s", err)
		return nil
	} else {
		return bs
	}
}

func unmarshalAndCheck(t *testing.T, bs []byte, obj any) {
	if err := unmarshal(bs, obj); err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
}

func TestRoundtrip(t *testing.T) {
	var businessObjDoc = []byte(`{
    "int_field": {"@int":"1234"},
    "long_field": {"@long":"123456"},
    "double_field": {"@double":"123.456"},
    "ptr_mod_field": {"@mod":"PtrFoo"},
    "mod_field": {"@mod":"Foo"},
    "ptr_ref_field": {"@ref":{"id":"1234","coll":{"@mod":"PtrFoo"}}},
    "ref_field": {"@ref":{"id":"1234","coll":{"@mod":"Foo:123"}}},
    "named_ref_field": {"@ref":{"name":"FooBar","coll":{"@mod":"Foo"}}},
    "set_field": {"@set":{"data":[0,1,2,3],"after":"foobarbaz"}},
    "doc_field": {"@doc": {
      "id": "1234",
      "coll": {"@mod":"Foo"},
      "ts": {"@time":"2023-02-28T18:10:10.00001Z"},
      "extra_field": "foobar"
    }},
    "named_doc_field": {"@doc": {
      "name": "mydoc",
      "coll": {"@mod":"Foo"},
      "ts": {"@time":"2023-02-28T18:10:10.00001Z"},
      "extra_field": "foobar"
    }},
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
  }`)

	obj := &BusinessObj{}
	unmarshalAndCheck(t, businessObjDoc, obj)

	bs := marshalAndCheck(t, obj)

	obj2 := &BusinessObj{}
	unmarshalAndCheck(t, bs, obj2)

	assert.Equal(t, obj, obj2)
}

func roundTripCheck(t *testing.T, test any, expected string) {
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
			if assert.JSONEq(t, `{"@time":"2023-02-28T18:10:10.00001Z"}`, string(bs)) {
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

func TestDecodingToInterface(t *testing.T) {
	var doc = []byte(`{
    "int_field": {"@int":"1234"},
    "long_field": {"@long":"123456"},
    "double_field": {"@double":"123.456"},
    "slice_field": [{"@mod":"Foo"}, {"@date":"2023-03-17"}],
    "obj_field": {"@object": {
      "string_field": "foobarbaz",
      "bool_field": true,
      "int_field": 1234,
      "double_field": 1234.567
    }}
  }`)

	var res any
	err := unmarshal(doc, &res)
	if assert.NoError(t, err) {
		rMap := res.(map[string]any)

		assert.Equal(t, int64(1234), rMap["int_field"].(int64))
		assert.Equal(t, int64(123456), rMap["long_field"].(int64))
		assert.Equal(t, float64(123.456), rMap["double_field"].(float64))

		sliceField := rMap["slice_field"].([]any)
		assert.Equal(t, &Module{"Foo"}, sliceField[0].(*Module))
		sliceDate := time.Date(2023, 03, 17, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, &sliceDate, sliceField[1].(*time.Time))

		objMap := rMap["obj_field"].(map[string]any)
		assert.Equal(t, "foobarbaz", objMap["string_field"].(string))
		assert.Equal(t, true, objMap["bool_field"].(bool))
		assert.Equal(t, float64(1234), objMap["int_field"].(float64))
		assert.Equal(t, 1234.567, objMap["double_field"].(float64))
	}
}

func TestEncodingFaunaStructs(t *testing.T) {
	t.Run("encodes Module", func(t *testing.T) {
		obj := Module{"Foo"}
		roundTripCheck(t, obj, `{"@mod":"Foo"}`)
	})

	t.Run("encodes Ref", func(t *testing.T) {
		obj := Ref{"1234", &Module{"Foo"}}
		roundTripCheck(t, obj, `{"@ref":{"id":"1234","coll":{"@mod":"Foo"}}}`)
	})

	t.Run("encodes NamedRef", func(t *testing.T) {
		obj := NamedRef{"Bar", &Module{"Foo"}}
		roundTripCheck(t, obj, `{"@ref":{"name":"Bar","coll":{"@mod":"Foo"}}}`)
	})

	t.Run("encodes Page", func(t *testing.T) {
		obj := Page{[]any{"0", "1", "2"}, "foobarbaz"}
		roundTripCheck(t, obj, `{"@set":{"data":["0","1","2"],"after":"foobarbaz"}}`)
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

func TestEncodingDocuments(t *testing.T) {
	t.Run("Document", func(t *testing.T) {
		type MyDoc struct {
			Document
			ExtraField1 string `fauna:"extra_field_1"`
			ExtraField2 string `fauna:"extra_field_2"`
		}

		doc := []byte(`{"@doc":{
      "id": "1234",
      "coll": {"@mod":"Foo"},
      "ts": {"@time":"2023-02-28T18:10:10.000010Z"},
      "extra_field_1": "foobar",
      "extra_field_2": "bazbuz"
    }}`)

		ts := time.Date(2023, 02, 28, 18, 10, 10, 10000, time.UTC)
		expected := MyDoc{
			Document: Document{
				ID:   "1234",
				Coll: &Module{"Foo"},
				TS:   &ts,
			},
			ExtraField1: "foobar",
			ExtraField2: "bazbuz",
		}

		var got MyDoc
		unmarshalAndCheck(t, doc, &got)
		assert.Equal(t, expected, got)

		encodedDoc := `{"@doc":{
    "id": "1234",
    "coll": {"@mod":"Foo"},
    "ts": {"@time":"2023-02-28T18:10:10.00001Z"},
    "extra_field_1": "foobar",
    "extra_field_2": "bazbuz"
  }}`
		bs := marshalAndCheck(t, expected)
		assert.JSONEq(t, encodedDoc, string(bs))
	})

	t.Run("NamedDocument", func(t *testing.T) {
		type MyDoc struct {
			NamedDocument
			ExtraField1 string `fauna:"extra_field_1"`
			ExtraField2 string `fauna:"extra_field_2"`
		}

		doc := []byte(`{"@doc":{
      "name": "mydoc",
      "coll": {"@mod":"Foo"},
      "ts": {"@time":"2023-02-28T18:10:10.000010Z"},
      "extra_field_1": "foobar",
      "extra_field_2": "bazbuz"
    }}`)

		ts := time.Date(2023, 02, 28, 18, 10, 10, 10000, time.UTC)
		expected := MyDoc{
			NamedDocument: NamedDocument{
				Name: "mydoc",
				Coll: &Module{"Foo"},
				TS:   &ts,
			},
			ExtraField1: "foobar",
			ExtraField2: "bazbuz",
		}

		var got MyDoc
		unmarshalAndCheck(t, doc, &got)
		assert.Equal(t, expected, got)

		encodedDoc := `{"@doc":{
      "name": "mydoc",
      "coll": {"@mod":"Foo"},
      "ts": {"@time":"2023-02-28T18:10:10.00001Z"},
      "extra_field_1": "foobar",
      "extra_field_2": "bazbuz"
    }}`
		bs := marshalAndCheck(t, expected)
		assert.JSONEq(t, encodedDoc, string(bs))
	})
}

func TestComposition(t *testing.T) {
	testDate := time.Date(2023, 2, 24, 0, 0, 0, 0, time.UTC)
	testDino := map[string]any{
		"name":      "Dino",
		"age":       0,
		"birthdate": testDate,
	}
	testInnerDino, err := FQL("let x = ${my_var}", map[string]any{"my_var": testDino})

	t.Run("template variable", func(t *testing.T) {
		encodedDoc := `{"fql":[
      "let x = ",
      {"value":{
        "age":{"@int":"0"},
        "birthdate":{"@time":"2023-02-24T00:00:00Z"},
        "name":"Dino"
      }}
    ]}`

		if assert.NoError(t, err) {
			bs := marshalAndCheck(t, testInnerDino)
			assert.JSONEq(t, encodedDoc, string(bs))
		}
	})

	t.Run("sub query", func(t *testing.T) {
		encodedDoc := `{"fql":[
      {"value":{
        "fql":[
          "let x = ",
          {"value":{
            "age":{"@int":"0"},
            "birthdate":{"@time":"2023-02-24T00:00:00Z"},
            "name":"Dino"
          }}
        ]
      }},
      "\nx { name }"
    ]}`

		if assert.NoError(t, err) {
			inner, err := FQL("${inner}\nx { name }", map[string]any{"inner": testInnerDino})
			if assert.NoError(t, err) {
				bs := marshalAndCheck(t, inner)
				assert.JSONEq(t, encodedDoc, string(bs))
			}
		}
	})
}
