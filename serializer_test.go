package fauna

import (
	"reflect"
	"testing"
	"time"
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
	IntField    int               `fauna:"int_field"`
	LongField   int64             `fauna:"long_field"`
	DoubleField float64           `fauna:"double_field"`
	DateField   time.Time         `fauna:"date_field,date"`
	TimeField   time.Time         `fauna:"time_field,time"`
	DocRefField DocumentReference `fauna:"doc_ref_field"`
	ModField    Module            `fauna:"mod_field"`
	ObjField    SubBusinessObj    `fauna:"obj_field"`

	IgnoredField string
}

func TestRoundtrip(t *testing.T) {
	var doc = `{
    "int_field": {"@int":"1234"},
    "long_field": {"@long":"123456"},
    "double_field": {"@double":"123.456"},
    "date_field": {"@date":"2000-01-01"},
    "time_field": {"@time":"2000-01-01T01:01:01.000Z"},
    "doc_ref_field": {"@doc":"Foo:123"},
    "mod_field": {"@mod":"foo"},
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
	if err := unmarshal([]byte(doc), obj); err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}

	bs, err := marshal(obj)
	if err != nil {
		t.Fatalf("failed to marshal: %s", err)
	}

	obj2 := &BusinessObj{}
	if err := unmarshal(bs, obj2); err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}

	if !reflect.DeepEqual(obj, obj2) {
		t.Fatal("objects did not round trip")
	}
}
