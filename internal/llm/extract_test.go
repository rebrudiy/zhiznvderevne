package llm

import "testing"

func TestExtractJSON(t *testing.T) {
	cases := map[string]string{
		`{"a":1}`:                         `{"a":1}`,
		"```json\n{\"a\":{\"b\":2}}\n```": `{"a":{"b":2}}`,
		`Вот пост: {"title":"x"} готово`:  `{"title":"x"}`,
		`no json here`:                    `no json here`,
		`} broken {`:                      `} broken {`,
	}
	for in, want := range cases {
		if got := extractJSON(in); got != want {
			t.Errorf("extractJSON(%q) = %q, want %q", in, got, want)
		}
	}
}
