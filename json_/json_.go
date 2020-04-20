package json_

import (
	"bytes"
	"encoding/json"
)

func Marshal(x interface{}) []byte {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return b
}

func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func Cast(x interface{}, y interface{}) error {
	b := Marshal(x)
	return Unmarshal(b, y)
}

func ToMap(x interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	err := Cast(x, &m)
	if err != nil {
		panic(err)
	}
	return m
}

func Format(b []byte) string {
	var buf bytes.Buffer
	err := json.Indent(&buf, b, "", "\t")
	if err != nil {
		return ""
	}
	return buf.String()
}
