package agentchat

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func parse(t *testing.T, raw string, s QuestionSchema) (QuestionCard, error) {
	t.Helper()
	return ParseQuestions(json.RawMessage(raw), s)
}

func TestParseQuestionsAcceptsAGroupedCard(t *testing.T) {
	card, err := parse(t, `{"title":"Harbour &amp; Co","description":" Two things. ","questions":[
		{"key":"role","label":"Target role","type":"text","placeholder":" e.g. Designer ","help":"One line","required":true},
		{"key":"mode","label":"Work arrangement","type":"select","options":["Remote","Hybrid","On site"]},
		{"key":"about","label":"About","type":"textarea"}]}`, QuestionSchema{})
	if err != nil {
		t.Fatal(err)
	}
	if card.Title != "Harbour & Co" || card.Description != "Two things." || len(card.Questions) != 3 {
		t.Fatalf("card = %+v", card)
	}
	q := card.Questions[0]
	if q.Placeholder != "e.g. Designer" || q.Help != "One line" || !q.Required || card.Token != "" {
		t.Errorf("question = %+v token=%q", q, card.Token)
	}
}

func TestParseQuestionsAllowedKeys(t *testing.T) {
	raw := `{"questions":[{"key":"role","label":"Role","type":"text"}]}`
	if _, err := parse(t, raw, QuestionSchema{AllowedKeys: []string{"role"}}); err != nil {
		t.Errorf("an allowed key: %v", err)
	}
	if _, err := parse(t, raw, QuestionSchema{AllowedKeys: []string{"name"}}); err == nil {
		t.Error("a key outside AllowedKeys must be rejected")
	}
	for _, key := range []string{"Role", "role-x", "", strings.Repeat("a", 41), "a b"} {
		body, _ := json.Marshal(map[string]any{"questions": []any{map[string]any{"key": key, "label": "L", "type": "text"}}})
		if _, err := ParseQuestions(body, QuestionSchema{}); err == nil {
			t.Errorf("key %q must be rejected when any key is allowed", key)
		}
	}
}

func TestParseQuestionsRejections(t *testing.T) {
	two := `{"key":"a","label":"A","type":"text"},{"key":"b","label":"B","type":"text"}`
	cases := map[string]string{
		"not json":                `[`,
		"no questions":            `{"questions":[]}`,
		"empty label":             `{"questions":[{"key":"a","label":" ","type":"text"}]}`,
		"unknown type":            `{"questions":[{"key":"a","label":"A","type":"date"}]}`,
		"duplicate key":           `{"questions":[` + two + `,{"key":"a","label":"A2","type":"text"}]}`,
		"choice without options":  `{"questions":[{"key":"a","label":"A","type":"select"}]}`,
		"one option":              `{"questions":[{"key":"a","label":"A","type":"select","options":["x"]}]}`,
		"duplicate option":        `{"questions":[{"key":"a","label":"A","type":"select","options":["Hybrid"," hybrid "]}]}`,
		"empty option":            `{"questions":[{"key":"a","label":"A","type":"select","options":["x"," "]}]}`,
		"non-text option":         `{"questions":[{"key":"a","label":"A","type":"select","options":["x",2]}]}`,
		"buttons beside another":  `{"questions":[{"key":"a","label":"A","type":"buttons","options":["x","y"]},{"key":"b","label":"B","type":"text"}]}`,
		"value outside options":   `{"questions":[{"key":"a","label":"A","type":"select","options":["x","y"],"value":"z"}]}`,
		"list on single choice":   `{"questions":[{"key":"a","label":"A","type":"select","options":["x","y"],"value":["x","y"]}]}`,
		"text for number":         `{"questions":[{"key":"a","label":"A","type":"number","value":"lots"}]}`,
		"negative number":         `{"questions":[{"key":"a","label":"A","type":"number","value":-1}]}`,
		"object value":            `{"questions":[{"key":"a","label":"A","type":"text","value":{"x":1}}]}`,
		"non-text list value":     `{"questions":[{"key":"a","label":"A","type":"multiselect","options":["x","y"],"value":[1]}]}`,
		"list on a text question": `{"questions":[{"key":"a","label":"A","type":"text","value":["x","y"]}]}`,
	}
	cases["twenty-one options"] = `{"questions":[{"key":"a","label":"A","type":"select","options":[` + strings.TrimSuffix(strings.Repeat(`"o",`, 21), ",") + `]}]}`
	for name, raw := range cases {
		if _, err := parse(t, raw, QuestionSchema{}); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
}

func TestParseQuestionsMaxQuestions(t *testing.T) {
	items := make([]string, 9)
	for i := range items {
		items[i] = `{"key":"k` + string(rune('a'+i)) + `","label":"L","type":"text"}`
	}
	nine := `{"questions":[` + strings.Join(items, ",") + `]}`
	if _, err := parse(t, nine, QuestionSchema{}); err == nil {
		t.Error("nine questions exceed the default of eight")
	}
	if _, err := parse(t, nine, QuestionSchema{MaxQuestions: 9}); err != nil {
		t.Errorf("MaxQuestions 9: %v", err)
	}
	eight := `{"questions":[` + strings.Join(items[:8], ",") + `]}`
	if _, err := parse(t, eight, QuestionSchema{MaxQuestions: 2}); err == nil {
		t.Error("eight questions exceed MaxQuestions 2")
	}
}

func TestParseQuestionsSnapsAndValidatesPrefill(t *testing.T) {
	card, err := parse(t, `{"questions":[
		{"key":"mode","label":"Where?","type":"select","options":["Remote","Hybrid"],"value":" hybrid "},
		{"key":"skills","label":"Skills","type":"multiselect","options":["Go","SQL"],"value":["go","SQL"]},
		{"key":"min","label":"Minimum","type":"number","value":5000},
		{"key":"company","label":"Company","type":"text","value":"Acme"},
		{"key":"blank","label":"Blank","type":"text","value":null}]}`, QuestionSchema{})
	if err != nil {
		t.Fatal(err)
	}
	qs := card.Questions
	if qs[0].Value != "Hybrid" || !reflect.DeepEqual(qs[1].Values, []string{"Go", "SQL"}) || qs[2].Value != "5000" || qs[3].Value != "Acme" || qs[4].Value != "" {
		t.Fatalf("prefill not normalised: %+v", qs)
	}
}

func TestQuestionToolSchema(t *testing.T) {
	open := QuestionToolSchema(QuestionSchema{})
	props := open["properties"].(map[string]any)
	q := props["questions"].(map[string]any)
	if q["maxItems"] != 8 {
		t.Errorf("default maxItems = %v", q["maxItems"])
	}
	key := q["items"].(map[string]any)["properties"].(map[string]any)["key"].(map[string]any)
	if key["pattern"] == nil || key["enum"] != nil {
		t.Errorf("any-key schema: %v", key)
	}
	limited := QuestionToolSchema(QuestionSchema{AllowedKeys: []string{"role", "name"}, MaxQuestions: 3})
	q = limited["properties"].(map[string]any)["questions"].(map[string]any)
	key = q["items"].(map[string]any)["properties"].(map[string]any)["key"].(map[string]any)
	if q["maxItems"] != 3 || !reflect.DeepEqual(key["enum"], []string{"role", "name"}) {
		t.Errorf("limited schema: maxItems=%v key=%v", q["maxItems"], key)
	}
	if _, err := json.Marshal(limited); err != nil {
		t.Errorf("the schema must be JSON: %v", err)
	}
}
