package fauna

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func Unmarshal(b []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()

	parser := jsonParser{decoder: decoder}
	result, parseErr := parser.parseNext()
	if parseErr != nil {
		return parseErr
	}

	// TODO: there has to be a more efficient way to do this
	//   I'm probably using the decoder wrong.
	jsonBytes, _ := json.Marshal(result)

	return json.Unmarshal(jsonBytes, v)
}

type jsonParser struct {
	decoder *json.Decoder
}

func (p *jsonParser) parseNext() (interface{}, error) {
	token, err := p.decoder.Token()
	if err != nil {
		return nil, err
	}

	switch token {
	case json.Delim('{'):
		return p.parseSpecialObject()
	case json.Delim('['):
		return p.parseArray()
	default:
		return p.parseLiteral(token)
	}
}

func (p *jsonParser) parseLiteral(token json.Token) (value interface{}, err error) {
	switch v := token.(type) {
	case string:
		value = fmt.Sprintf("%v", v)
	case bool:
		value, _ = strconv.ParseBool(fmt.Sprintf("%v", v))
	case json.Number:
		value, err = p.parseJSONNumber(v)
	case nil:
		value = nil
	default:
		err = fmt.Errorf("unknown literal: %v", token)
	}

	return
}

func (p *jsonParser) parseSpecialObject() (interface{}, error) {
	if !p.hasMore() {
		return nil, nil
	}

	if firstKey, err := p.readString(); err == nil {
		switch firstKey {
		case "@date":
			return p.parseDate("2006-01-02", func(t time.Time) time.Time { return t })
		case "@time":
			return p.parseDate("2006-01-02T15:04:05.999Z", func(t time.Time) time.Time { return t })
		case "@int":
			return p.parseInt32()
		case "@long":
			return p.parseInt64()
		case "@double":
			return p.parseDouble()
		default:
			return p.parseObject(firstKey)
		}
	}

	return nil, nil
}

func (p *jsonParser) parseObject(firstKey string) (map[string]interface{}, error) {
	object := make(map[string]interface{})

	if key := firstKey; key != "" {
		for {
			if value, err := p.parseNext(); err == nil {
				object[key] = value

				if !p.hasMore() {
					break
				}

				if key, err = p.readString(); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	return object, nil
}

func (p *jsonParser) parseInt32() (value int32, err error) {
	var str string
	if str, err = p.readSingleString(); err == nil {
		var intValue int
		intValue, err = strconv.Atoi(str)
		value = int32(intValue)
	}

	return
}

func (p *jsonParser) parseInt64() (value int64, err error) {
	var str string
	if str, err = p.readSingleString(); err == nil {
		value, err = strconv.ParseInt(str, 10, 64)
	}

	return
}

func (p *jsonParser) parseDouble() (value float64, err error) {
	var str string
	if str, err = p.readSingleString(); err == nil {
		value, err = strconv.ParseFloat(str, 16)
	}

	return
}

func (p *jsonParser) parseBytes() (value []byte, err error) {
	var encoded string

	if encoded, err = p.readSingleString(); err == nil {
		bytesV, bytesErr := base64.StdEncoding.DecodeString(encoded)
		if bytesErr == nil {
			value = bytesV
		}
	}

	return
}

func (p *jsonParser) parseQuery() (value json.RawMessage, err error) {
	var lambda json.RawMessage

	if err = p.decoder.Decode(&lambda); err == nil {
		value = lambda
	}

	var token json.Token
	if token, err = p.decoder.Token(); err == nil {
		if token != json.Delim('}') {
			err = fmt.Errorf("end of object: %v", token)
		}
	}

	return
}

func (p *jsonParser) parseArray() ([]interface{}, error) {
	var array []interface{}

	for {
		if !p.hasMore() {
			break
		}

		if value, err := p.parseNext(); err == nil {
			array = append(array, value)
		} else {
			return nil, err
		}
	}

	return array, nil
}

func (p *jsonParser) parseDate(format string, fn func(t time.Time) time.Time) (value time.Time, err error) {
	var str string

	if str, err = p.readSingleString(); err == nil {
		value, err = p.parseStrTime(str, format, fn)
	}

	return
}

func (p *jsonParser) parseStrTime(raw string, format string, fn func(time.Time) time.Time) (value time.Time, err error) {
	var t time.Time

	if t, err = time.Parse(format, raw); err == nil {
		value = fn(t)
	}

	return
}

func (p *jsonParser) parseJSONNumber(number json.Number) (interface{}, error) {
	var err error

	if strings.Contains(number.String(), ".") {
		var n float64
		if n, err = number.Float64(); err == nil {
			return n, nil
		}
	} else {
		var n int64
		if n, err = number.Int64(); err == nil {
			return n, nil
		}
	}

	return nil, err
}

func (p *jsonParser) readSingleString() (str string, err error) {
	if str, err = p.readString(); err == nil {
		err = p.ensureNoMoreTokens()
	}

	return
}

func (p *jsonParser) readString() (str string, err error) {
	var token json.Token
	var ok bool

	if token, err = p.decoder.Token(); err == nil {
		if str, ok = token.(string); !ok {
			err = fmt.Errorf("a string: %v", token)
		}
	}

	return
}

func (p *jsonParser) ensureNoMoreTokens() error {
	if p.hasMore() {
		token, _ := p.decoder.Token()
		return fmt.Errorf("end of array or object: %v", token)
	}

	return nil
}

func (p *jsonParser) hasMore() bool {
	if !p.decoder.More() {
		_, _ = p.decoder.Token()
		return false
	}

	return true
}
