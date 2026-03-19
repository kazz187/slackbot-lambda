package slackbot

import "encoding/json"

func MustJSONMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func MustJSONUnmarshal[T any](data string) T {
	var v T
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		panic(err)
	}
	return v
}
