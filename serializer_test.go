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

	IgnoredField2 string `fauna:"-"`
}

type DocBusinessObj struct {
	Document
	ExtraField string `fauna:"extra_field"`
}

type NamedDocBusinessObj struct {
	NamedDocument
	ExtraField string `fauna:"extra_field"`
}

type NullDocBusinessObj struct {
	NullDocument
}

type NullNamedDocBusinessObj struct {
	NullNamedDocument
}

type BusinessObj struct {
	IntField          int                     `fauna:"int_field"`
	LongField         int64                   `fauna:"long_field"`
	DoubleField       float64                 `fauna:"double_field"`
	PtrModField       *Module                 `fauna:"ptr_mod_field"`
	ModField          Module                  `fauna:"mod_field"`
	PtrRefField       *Ref                    `fauna:"ptr_ref_field"`
	RefField          Ref                     `fauna:"ref_field"`
	NamedRefField     NamedRef                `fauna:"named_ref_field"`
	SetField          Page                    `fauna:"set_field"`
	ObjField          SubBusinessObj          `fauna:"obj_field"`
	DocField          DocBusinessObj          `fauna:"doc_field"`
	NamedDocField     NamedDocBusinessObj     `fauna:"named_doc_field"`
	NullDocField      NullDocBusinessObj      `fauna:"nulldoc_field"`
	NullNamedDocField NullNamedDocBusinessObj `fauna:"nulldoc_named_field"`

	IgnoredField string `fauna:"-"`
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

func encodeCheck(t *testing.T, test any, expected string) {
	bs := marshalAndCheck(t, test)
	assert.JSONEq(t, expected, string(bs))
}

func decodeCheck(t *testing.T, test string, expected any) {
	b := []byte(test)
	decoded := reflect.New(reflect.TypeOf(expected)).Interface()
	unmarshalAndCheck(t, b, decoded)
	dec := reflect.Indirect(reflect.ValueOf(decoded)).Interface()
	assert.EqualValues(t, expected, dec)
}

func TestEncodingPrimitives(t *testing.T) {
	t.Run("encode string", func(t *testing.T) {
		encodeCheck(t, "foo", `{ "value": "foo"}`)
		decodeCheck(t, `"foo"`, "foo")
	})

	t.Run("encode bools", func(t *testing.T) {
		encodeCheck(t, true, `{"value": true}`)
		encodeCheck(t, false, `{"value": false}`)

		decodeCheck(t, `true`, true)
		decodeCheck(t, `false`, false)
	})

	t.Run("encode ints", func(t *testing.T) {
		encodeCheck(t, int(10), `{"value":{"@int":"10"}}`)
		encodeCheck(t, int8(10), `{"value":{"@int":"10"}}`)
		encodeCheck(t, int16(10), `{"value":{"@int":"10"}}`)
		encodeCheck(t, int32(10), `{"value":{"@int":"10"}}`)
		encodeCheck(t, int(-10), `{"value":{"@int":"-10"}}`)
		encodeCheck(t, int8(-10), `{"value":{"@int":"-10"}}`)
		encodeCheck(t, int16(-10), `{"value":{"@int":"-10"}}`)
		encodeCheck(t, int32(-10), `{"value":{"@int":"-10"}}`)
		encodeCheck(t, 2147483647, `{"value":{"@int":"2147483647"}}`)
		encodeCheck(t, -2147483648, `{"value":{"@int":"-2147483648"}}`)

		decodeCheck(t, `{"@int":"10"}`, int(10))
		decodeCheck(t, `{"@int":"10"}`, int8(10))
		decodeCheck(t, `{"@int":"10"}`, int16(10))
		decodeCheck(t, `{"@int":"10"}`, int32(10))
		decodeCheck(t, `{"@int":"-10"}`, int(-10))
		decodeCheck(t, `{"@int":"-10"}`, int8(-10))
		decodeCheck(t, `{"@int":"-10"}`, int16(-10))
		decodeCheck(t, `{"@int":"-10"}`, int32(-10))
		decodeCheck(t, `{"@int":"2147483647"}`, 2147483647)
		decodeCheck(t, `{"@int":"-2147483648"}`, -2147483648)
	})

	t.Run("encode longs", func(t *testing.T) {
		encodeCheck(t, 2147483648, `{"value":{"@long":"2147483648"}}`)
		encodeCheck(t, -2147483649, `{"value":{"@long":"-2147483649"}}`)
		encodeCheck(t, 9223372036854775807, `{"value":{"@long":"9223372036854775807"}}`)
		encodeCheck(t, -9223372036854775808, `{"value":{"@long":"-9223372036854775808"}}`)

		decodeCheck(t, `{"@long":"2147483648"}`, 2147483648)
		decodeCheck(t, `{"@long":"-2147483649"}`, -2147483649)
		decodeCheck(t, `{"@long":"9223372036854775807"}`, 9223372036854775807)
		decodeCheck(t, `{"@long":"-9223372036854775808"}`, -9223372036854775808)
	})

	t.Run("fail on numbers that are too big", func(t *testing.T) {
		tooLarge := uint(9223372036854775808)
		_, tooLargeErr := marshal(tooLarge)
		assert.Error(t, tooLargeErr)
	})

	t.Run("encode floats", func(t *testing.T) {
		encodeCheck(t, 100.0, `{"value":{"@double":"100"}}`)
		encodeCheck(t, -100.1, `{"value":{"@double":"-100.1"}}`)
		encodeCheck(t, 9.999999999999, `{"value":{"@double":"9.999999999999"}}`)

		decodeCheck(t, `{"@double":"100"}`, 100.0)
		decodeCheck(t, `{"@double":"-100.1"}`, -100.1)
		decodeCheck(t, `{"@double":"9.999999999999"}`, 9.999999999999)
	})

	t.Run("encode nil", func(t *testing.T) {
		encodeCheck(t, nil, `{"value": null}`)

		var decoded *string
		unmarshalAndCheck(t, []byte("null"), &decoded)
		assert.Nil(t, decoded)
	})
}

func TestEncodingTime(t *testing.T) {
	t.Run("encodes time as @time", func(t *testing.T) {
		if tz, err := time.LoadLocation("America/Los_Angeles"); assert.NoError(t, err) {
			bs := marshalAndCheck(t, time.Date(2023, 02, 28, 10, 10, 10, 10000, tz))
			if assert.JSONEq(t, `{"value": {"@time":"2023-02-28T18:10:10.00001Z"}}`, string(bs)) {
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
		encodeCheck(t, obj, `{"object": {"d_field": { "value": {"@date":"2023-02-28"}}}}`)
		decodeCheck(t, `{"d_field": {"@date":"2023-02-28"}}`, obj)
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
		encodeCheck(t, obj, `{"value":{"@mod":"Foo"}}`)
		decodeCheck(t, `{"@mod":"Foo"}`, obj)
	})

	t.Run("encodes Ref", func(t *testing.T) {
		obj := Ref{"1234", &Module{"Foo"}}
		encodeCheck(t, obj, `{"value":{"@ref":{"id":"1234","coll":{"@mod":"Foo"}}}}`)
		decodeCheck(t, `{"@ref":{"id":"1234","coll":{"@mod":"Foo"}}}`, obj)
	})

	t.Run("encodes NamedRef", func(t *testing.T) {
		obj := NamedRef{"Bar", &Module{"Foo"}}
		encodeCheck(t, obj, `{"value":{"@ref":{"name":"Bar","coll":{"@mod":"Foo"}}}}`)
		decodeCheck(t, `{"@ref":{"name":"Bar","coll":{"@mod":"Foo"}}}`, obj)
	})

	t.Run("encodes Page", func(t *testing.T) {
		obj := Page{[]any{"0", "1", "2"}, "foobarbaz"}
		encodeCheck(t, obj, `{"value":{"@set":{"data":["0","1","2"],"after":"foobarbaz"}}}`)
		decodeCheck(t, `{"@set":{"data":["0","1","2"],"after":"foobarbaz"}}`, obj)
	})

	t.Run("encode NullDoc", func(t *testing.T) {
		obj := NullDocument{Cause: "Foo", Ref: &Ref{ID: "1234", Coll: &Module{"Foo"}}}
		encodeCheck(t, obj, `{"value": {"@ref":{"id":"1234","coll":{"@mod":"Foo"}}}}`)
		decodeCheck(t, `{"cause": "Foo", "ref": {"@ref":{"id":"1234","coll":{"@mod":"Foo"}}}}`, obj)
	})

	t.Run("decodes data-less set", func(t *testing.T) {
		bs := []byte(`{"@set":"foobarbaz"}`)
		var set any
		unmarshalAndCheck(t, bs, &set)
		if page, ok := set.(*Page); assert.True(t, ok) {
			assert.Nil(t, page.Data)
			assert.Equal(t, "foobarbaz", page.After)
		}
	})
}

func TestEncodingStructs(t *testing.T) {
	t.Run("encodes using struct field names", func(t *testing.T) {
		obj := struct {
			Field string
		}{"foo"}

		encodeCheck(t, obj, `{"object": {"Field":{"value": "foo"}}}`)
		decodeCheck(t, `{"Field":"foo"}`, obj)
	})

	t.Run("encodes using configured field names", func(t *testing.T) {
		obj := struct {
			Field string `fauna:"field_name"`
		}{"foo"}

		encodeCheck(t, obj, `{"object": {"field_name":{"value": "foo"}}}`)
		decodeCheck(t, `{"field_name":"foo"}`, obj)
	})

	t.Run("encodes hinted types without configured name", func(t *testing.T) {
		obj := struct {
			Field time.Time `fauna:",date"`
		}{
			Field: time.Date(2023, 02, 28, 0, 0, 0, 0, time.UTC),
		}
		encodeCheck(t, obj, `{"object":{"Field":{"value":{"@date":"2023-02-28"}}}}`)
		decodeCheck(t, `{"Field":{"@date":"2023-02-28"}}`, obj)
	})

	t.Run("encodes hinted types with configured name", func(t *testing.T) {
		obj := struct {
			Field time.Time `fauna:"field_name,date"`
		}{
			Field: time.Date(2023, 02, 28, 0, 0, 0, 0, time.UTC),
		}

		encodeCheck(t, obj, `{"object": {"field_name":{"value": {"@date":"2023-02-28"}}}}`)
		decodeCheck(t, `{"field_name":{"@date":"2023-02-28"}}`, obj)
	})

	t.Run("ignores fields", func(t *testing.T) {
		obj := struct {
			Field        string
			IgnoredField string `fauna:"-"`
		}{
			Field:        "foo",
			IgnoredField: "",
		}
		encodeCheck(t, obj, `{"object": {"Field":{"value": "foo"}}}`)
		decodeCheck(t, `{"Field":"foo"}`, obj)
	})

	t.Run("encodes nested fields", func(t *testing.T) {
		var obj struct {
			GrandParent struct {
				Parent struct {
					Child   string
					Sibling string
				}
			}
		}
		obj.GrandParent.Parent.Child = "foo"
		obj.GrandParent.Parent.Sibling = "bar"

		encodeCheck(t, obj, `{"object":{"GrandParent":{"object":{"Parent":{"object":{"Child":{"value": "foo"},"Sibling":{"value":"bar"}}}}}}}`)
		decodeCheck(t, `{"GrandParent":{"Parent":{"Child":"foo","Sibling":"bar"}}}`, obj)
	})
}

func TestEncodingPointers(t *testing.T) {
	type checkStruct struct {
		Field string
	}

	var obj struct {
		NilPtrField *checkStruct
		PtrField    *checkStruct
		Field       checkStruct
	}
	obj.NilPtrField = nil
	obj.PtrField = &checkStruct{"foo"}
	obj.Field = checkStruct{"bar"}
	encodeCheck(t, obj, `{"object":{"NilPtrField":null,"PtrField":{"object":{"Field":{"value":"foo"}}},"Field":{"object":{"Field":{"value":"bar"}}}}}`)
	decodeCheck(t, `{"NilPtrField":null,"PtrField":{"Field":"foo"},"Field":{"Field":"bar"}}`, obj)
}

func TestEncodingObject(t *testing.T) {
	t.Run("object has @object key", func(t *testing.T) {
		test := map[string]int{"@object": 10}
		encodeCheck(t, test, `{"object":{"@object":{"value":{"@int":"10"}}}}`)
	})

	t.Run("object has inner conflicting @int key", func(t *testing.T) {
		test := map[string]map[string]string{"@object": {"@int": "bar"}}
		encodeCheck(t, test, `{"object":{"@object":{"object":{"@int":{"value":"bar"}}}}}`)
		decodeCheck(t, `{"@object":{"@object":{"@object":{"@int":"bar"}}}}`, test)
	})

	t.Run("object has inner conflicting @object key", func(t *testing.T) {
		test := map[string]map[string]string{"@object": {"@object": "bar"}}
		encodeCheck(t, test, `{"object":{"@object":{"object":{"@object":{"value":"bar"}}}}}`)
		decodeCheck(t, `{"@object":{"@object":{"@object":{"@object":"bar"}}}}`, test)
	})

	t.Run("object has multiple conflicting type keys", func(t *testing.T) {
		test := map[string]string{"@int": "foo", "@double": "bar"}
		encodeCheck(t, test, `{"object":{"@int":{"value":"foo"},"@double":{"value":"bar"}}}`)
		decodeCheck(t, `{"@object":{"@int":"foo","@double":"bar"}}`, test)
	})

	t.Run("object has mixed keys with a conflict", func(t *testing.T) {
		test := map[string]string{"@int": "foo", "bar": "buz"}
		encodeCheck(t, test, `{"object":{"@int":{"value":"foo"},"bar":{"value":"buz"}}}`)
		decodeCheck(t, `{"@object":{"@int":"foo","bar":"buz"}}`, test)
	})

	t.Run("object has nested conflicting keys", func(t *testing.T) {
		test := map[string]map[string]map[string]map[string]int{"@int": {"@date": {"@time": {"@long": 10}}}}
		encodeCheck(t, test, `{"object":{"@int":{"object":{"@date":{"object":{"@time":{"object":{"@long":{"value":{"@int":"10"}}}}}}}}}}`)
		decodeCheck(t, `{"@object":{"@int":{"@object":{"@date":{"@object":{"@time":{"@object":{"@long":{"@int":"10"}}}}}}}}}`, test)
	})

	t.Run("object has non-conflicting keys", func(t *testing.T) {
		test := map[string]int{"@foo": 10}
		encodeCheck(t, test, `{"object":{"@foo":{"value":{"@int":"10"}}}}`)
		decodeCheck(t, `{"@foo":{"@int":"10"}}`, test)
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

		encodedDoc := `{"value":{"@ref":{"id": "1234","coll": {"@mod":"Foo"}}}}`
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

		encodedDoc := `{"value":{"@ref":{"name":"mydoc","coll":{"@mod":"Foo"}}}}`
		bs := marshalAndCheck(t, expected)
		assert.JSONEq(t, encodedDoc, string(bs))
	})

	t.Run("Raw Document", func(t *testing.T) {
		doc := []byte(`{"@doc":{
      "id": "1234",
      "coll": {"@mod":"Foo"},
      "ts": {"@time":"2023-02-28T18:10:10.000010Z"}
    }}`)

		ts := time.Date(2023, 02, 28, 18, 10, 10, 10000, time.UTC)
		expected := Document{
			ID:   "1234",
			Coll: &Module{"Foo"},
			TS:   &ts,
		}

		var got Document
		unmarshalAndCheck(t, doc, &got)
		assert.Equal(t, expected, got)

		encodedDoc := `{"value":{"@ref": {"id": "1234","coll": {"@mod":"Foo"}}}}`
		bs := marshalAndCheck(t, expected)
		assert.JSONEq(t, encodedDoc, string(bs))
	})

	t.Run("Raw NamedDocument", func(t *testing.T) {
		doc := []byte(`{"@doc":{
      "name": "mydoc",
      "coll": {"@mod":"Foo"},
      "ts": {"@time":"2023-02-28T18:10:10.000010Z"}
    }}`)

		ts := time.Date(2023, 02, 28, 18, 10, 10, 10000, time.UTC)
		expected := NamedDocument{
			Name: "mydoc",
			Coll: &Module{"Foo"},
			TS:   &ts,
		}

		var got NamedDocument
		unmarshalAndCheck(t, doc, &got)
		assert.Equal(t, expected, got)

		encodedDoc := `{"value":{"@ref": {"name": "mydoc","coll": {"@mod":"Foo"}}}}`
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
      {"object":{
        "age":{"value": {"@int":"0"}},
        "birthdate":{"value":{"@time":"2023-02-24T00:00:00Z"}},
        "name":{"value":"Dino"}
      }}
    ]}`

		if assert.NoError(t, err) {
			bs := marshalAndCheck(t, testInnerDino)
			assert.JSONEq(t, encodedDoc, string(bs))
		}
	})

	t.Run("sub query", func(t *testing.T) {
		encodedDoc := `{"fql":[
      {"fql":[
        "let x = ",
        {"object":{
          "age":{"value":{"@int":"0"}},
          "birthdate":{"value":{"@time":"2023-02-24T00:00:00Z"}},
          "name":{"value":"Dino"}
        }}
      ]},
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

	t.Run("sub queries embedded in slices and objects", func(t *testing.T) {

		aMap := map[string]any{
			"q1": testInnerDino,
		}
		aSlice := []any{
			testInnerDino,
			aMap,
		}

		encodedDoc := `{"fql":[{
			"array": [
				{"fql":[
					"let x = ",
					{"object":{
					  "age":{"value":{"@int":"0"}},
					  "birthdate":{"value":{"@time":"2023-02-24T00:00:00Z"}},
					  "name":{"value":"Dino"}
					}}
                ]},
				{"object":{"q1":
					{"fql":[
						"let x = ",
						{"object":{
						  "age":{"value":{"@int":"0"}},
						  "birthdate":{"value":{"@time":"2023-02-24T00:00:00Z"}},
						  "name":{"value":"Dino"}
						}}
					]}
				}}
			]
		}]}`

		if assert.NoError(t, err) {
			inner, err := FQL("${inner}", map[string]any{"inner": aSlice})
			if assert.NoError(t, err) {
				bs := marshalAndCheck(t, inner)
				assert.JSONEq(t, encodedDoc, string(bs))
			}
		}
	})
}
