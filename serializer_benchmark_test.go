package fauna

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkUnmarshalInt(b *testing.B) {
	v := []byte(`{"@int":"1234"}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res int
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalLong(b *testing.B) {
	v := []byte(`{"@long":"4294967297"}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res int64
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalDouble(b *testing.B) {
	v := []byte(`{"@double":"123.456"}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res float64
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalBool(b *testing.B) {
	v := []byte(`true`)
	var err error
	for i := 0; i < b.N; i++ {
		var res bool
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalTime(b *testing.B) {
	v := []byte(`{"@time":"2023-02-28T18:10:10.00001Z"}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res time.Time
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalModulePointer(b *testing.B) {
	v := []byte(`{"@mod":"PtrFoo"}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res *Module
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalModule(b *testing.B) {
	v := []byte(`{"@mod":"PtrFoo"}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res Module
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalRefPointer(b *testing.B) {
	v := []byte(`{"@ref":{"id":"1234","coll":{"@mod":"PtrFoo"}}}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res *Ref
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalRef(b *testing.B) {
	v := []byte(`{"@ref":{"id":"1234","coll":{"@mod":"Foo"}}}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res Ref
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalNamedRefPointer(b *testing.B) {
	v := []byte(`{"@ref":{"name":"BarPtr","coll":{"@mod":"FooPtr"}}}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res *NamedRef
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalNamedRef(b *testing.B) {
	v := []byte(`{"@ref":{"name":"Bar","coll":{"@mod":"Foo"}}}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res NamedRef
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalSet(b *testing.B) {
	v := []byte(`{"@set":{"data":[0,1,2,3],"after":"foobarbaz"}}`)
	var err error
	for i := 0; i < b.N; i++ {
		var res Page
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalDocument(b *testing.B) {
	v := []byte(`{"@doc": {
		"id": "1234",
			"coll": {"@mod":"Foo"},
		"ts": {"@time":"2023-02-28T18:10:10.00001Z"},
		"extra_field": "foobar"
	}}`)

	var err error
	for i := 0; i < b.N; i++ {
		var res DocBusinessObj
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalNamedDocument(b *testing.B) {
	v := []byte(`{"@doc": {
      "name": "mydoc",
      "coll": {"@mod":"Foo"},
      "ts": {"@time":"2023-02-28T18:10:10.00001Z"},
      "extra_field": "foobar"
    }}`)

	var err error
	for i := 0; i < b.N; i++ {
		var res NamedDocBusinessObj
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalNullDocument(b *testing.B) {
	v := []byte(`{"@ref":{"id":"1234","coll":{"@mod":"Foo:123"},"exists":false,"cause":"foobar"}}`)

	var err error
	for i := 0; i < b.N; i++ {
		var res NullDocument
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalNamedNullDocument(b *testing.B) {
	v := []byte(`{"@ref":{"name":"FooBar","coll":{"@mod":"Foo"},"exists":false,"cause":"foobar"}}`)

	var err error
	for i := 0; i < b.N; i++ {
		var res NullNamedDocument
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalObject(b *testing.B) {
	v := []byte(`{"@object": {
      "string_field": "foobarbaz",
      "bool_field": true,
      "map_field": {"foo":"bar","baz":"buz"},
      "single_key_map_field": {"foo":"bar"},
      "slice_field": [1,2,3,4],
      "tuple_field": ["one",2,3.0],
      "int_field": 1234,
      "double_field": 1234.567
    }}`)

	var err error
	for i := 0; i < b.N; i++ {
		var res SubBusinessObj
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalSlice(b *testing.B) {
	v := []byte(`[1,2,3,4]`)

	var err error
	for i := 0; i < b.N; i++ {
		var res []int
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}

func BenchmarkUnmarshalBytes(b *testing.B) {
	v := []byte(`{"@bytes":"SGVsbG8sIGZyb20gRmF1bmEh"}`)

	var err error
	for i := 0; i < b.N; i++ {
		var res []byte
		err = unmarshal(v, &res)
		assert.NoError(b, err)
	}
}
