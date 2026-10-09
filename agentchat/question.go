package agentchat

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	defaultMaxQuestions = 8
	maxOptions          = 20
)

var keyPattern = regexp.MustCompile(`^[a-z0-9_]{1,40}$`)

// Question types a card can ask. buttons, select and multiselect are choices
// and need Options; a buttons question is the only question of its card.
const (
	TypeButtons     = "buttons"
	TypeSelect      = "select"
	TypeMultiselect = "multiselect"
	TypeNumber      = "number"
	TypeText        = "text"
	TypeTextarea    = "textarea"
)

func isChoice(kind string) bool {
	return kind == TypeButtons || kind == TypeSelect || kind == TypeMultiselect
}

// Question is one field of an agent-driven question card.
//
//	Key — the field name without its q_ prefix.
//	Type — one of the Type constants.
//	Value, Values — the agent's suggested answer (Values for multiselect), so
//	    the user can mostly confirm. Choices snap to the exact option text.
type Question struct {
	Key, Label, Help, Type, Placeholder, Value string
	Options, Values                            []string
	Required                                   bool
}

// QuestionCard is a validated set of questions. Token identifies this exact
// card: the app sets it (for example a hash of the session and the questions)
// and refuses answers that come back with a different one.
type QuestionCard struct {
	Token, Title, Description string
	Questions                 []Question
}

// QuestionSchema is the app's policy for the questions a model may ask.
//
//	AllowedKeys — the only keys allowed; nil allows any [a-z0-9_]{1,40}.
//	MaxQuestions — the most questions on one card (default 8).
type QuestionSchema struct {
	AllowedKeys  []string
	MaxQuestions int
}

func (s QuestionSchema) max() int {
	if s.MaxQuestions > 0 {
		return s.MaxQuestions
	}
	return defaultMaxQuestions
}

func (s QuestionSchema) allows(key string) bool {
	if s.AllowedKeys == nil {
		return keyPattern.MatchString(key)
	}
	return slices.Contains(s.AllowedKeys, key)
}

type rawCard struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Questions   []rawQuestion `json:"questions"`
}

type rawQuestion struct {
	Key         string          `json:"key"`
	Label       string          `json:"label"`
	Type        string          `json:"type"`
	Help        string          `json:"help"`
	Placeholder string          `json:"placeholder"`
	Options     []any           `json:"options"`
	Value       json.RawMessage `json:"value"`
	Required    bool            `json:"required"`
}

// ParseQuestions validates the arguments of the app's question tool call
// (the shape QuestionToolSchema describes) and returns the card to render.
// Anything outside the schema is an error to report back to the model: an
// unknown key, a repeated key or option, a choice without 2 to 20 options,
// a buttons question beside others, or a suggested value that is not one of
// the options or not a non-negative number. The returned card has no Token.
func ParseQuestions(raw json.RawMessage, s QuestionSchema) (QuestionCard, error) {
	var in rawCard
	if err := json.Unmarshal(raw, &in); err != nil {
		return QuestionCard{}, errors.New("questions must be an object with a questions list")
	}
	if n := len(in.Questions); n < 1 || n > s.max() {
		return QuestionCard{}, fmt.Errorf("questions must contain between 1 and %d items", s.max())
	}
	// Models sometimes write HTML entities ("Harbour &amp; Co") into copy; templ
	// escapes on output, so show the characters.
	card := QuestionCard{Title: unescape(in.Title), Description: unescape(in.Description)}
	seen := make(map[string]bool, len(in.Questions))
	for _, r := range in.Questions {
		q := Question{Key: r.Key, Label: unescape(r.Label), Type: r.Type, Help: unescape(r.Help), Placeholder: strings.TrimSpace(r.Placeholder), Required: r.Required}
		if !s.allows(q.Key) || q.Label == "" {
			return QuestionCard{}, errors.New("question key must be supported and label cannot be empty")
		}
		if seen[q.Key] {
			return QuestionCard{}, fmt.Errorf("question key %q is repeated", q.Key)
		}
		seen[q.Key] = true
		switch q.Type {
		case TypeButtons, TypeSelect, TypeMultiselect, TypeNumber, TypeText, TypeTextarea:
		default:
			return QuestionCard{}, errors.New("question type must be buttons, select, multiselect, number, text or textarea")
		}
		if q.Type == TypeButtons && len(in.Questions) != 1 {
			return QuestionCard{}, errors.New("immediate buttons require a single-question group")
		}
		if isChoice(q.Type) {
			options, err := cleanOptions(q.Key, r.Options)
			if err != nil {
				return QuestionCard{}, err
			}
			q.Options = options
		}
		if err := setPrefill(&q, r.Value); err != nil {
			return QuestionCard{}, err
		}
		card.Questions = append(card.Questions, q)
	}
	return card, nil
}

func unescape(s string) string { return strings.TrimSpace(html.UnescapeString(s)) }

func cleanOptions(key string, raw []any) ([]string, error) {
	if len(raw) < 2 || len(raw) > maxOptions {
		return nil, fmt.Errorf("choice questions need between 2 and %d options", maxOptions)
	}
	seen := make(map[string]bool, len(raw))
	options := make([]string, 0, len(raw))
	for _, o := range raw {
		s, ok := o.(string)
		if s = strings.TrimSpace(s); !ok || s == "" {
			return nil, errors.New("question options must be nonempty text")
		}
		if seen[strings.ToLower(s)] {
			return nil, fmt.Errorf("choice question %q contains a repeated option", key)
		}
		seen[strings.ToLower(s)] = true
		options = append(options, s)
	}
	return options, nil
}

// setPrefill validates the suggested answer against the question type and,
// for choices, snaps it to the exact option text.
func setPrefill(q *Question, raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var picked []string
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("value for %q must be text or a list of text", q.Key)
	}
	switch v := v.(type) {
	case string:
		if s := strings.TrimSpace(v); s != "" {
			picked = []string{s}
		}
	case float64:
		picked = []string{strconv.FormatFloat(v, 'f', -1, 64)}
	case []any:
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return fmt.Errorf("value for %q must be text or a list of text", q.Key)
			}
			if s = strings.TrimSpace(s); s != "" {
				picked = append(picked, s)
			}
		}
	default:
		return fmt.Errorf("value for %q must be text or a list of text", q.Key)
	}
	if len(picked) == 0 {
		return nil
	}
	if isChoice(q.Type) {
		for i, p := range picked {
			option, ok := matchOption(q.Options, p)
			if !ok {
				return fmt.Errorf("value %q for %q is not one of its options", p, q.Key)
			}
			picked[i] = option
		}
	}
	if q.Type == TypeMultiselect {
		q.Values = picked
		return nil
	}
	if len(picked) != 1 {
		return fmt.Errorf("%q takes a single value", q.Key)
	}
	if q.Type == TypeNumber {
		n, err := strconv.ParseFloat(picked[0], 64)
		if err != nil || n < 0 || math.IsNaN(n) || math.IsInf(n, 0) {
			return fmt.Errorf("value for %q must be a number", q.Key)
		}
	}
	q.Value = picked[0]
	return nil
}

func matchOption(options []string, value string) (string, bool) {
	for _, o := range options {
		if strings.EqualFold(o, value) {
			return o, true
		}
	}
	return "", false
}

// QuestionToolSchema returns the JSON schema of the question tool's
// arguments, for the app's tool definition; ParseQuestions validates the call.
func QuestionToolSchema(s QuestionSchema) map[string]any {
	key := map[string]any{"type": "string", "pattern": keyPattern.String()}
	if s.AllowedKeys != nil {
		key = map[string]any{"type": "string", "enum": s.AllowedKeys}
	}
	question := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"key", "label", "type"},
		"properties": map[string]any{
			"key":         key,
			"label":       map[string]any{"type": "string"},
			"type":        map[string]any{"type": "string", "enum": []string{TypeButtons, TypeSelect, TypeMultiselect, TypeNumber, TypeText, TypeTextarea}},
			"options":     map[string]any{"type": "array", "maxItems": maxOptions, "items": map[string]any{"type": "string"}},
			"help":        map[string]any{"type": "string"},
			"placeholder": map[string]any{"type": "string"},
			"required":    map[string]any{"type": "boolean"},
			"value": map[string]any{
				"type": []string{"string", "number", "array"}, "items": map[string]any{"type": "string"},
				"description": "The suggested answer from what you already know, so the user can just confirm. For choices it must be one of the options (an array of options for multiselect).",
			},
		},
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"questions"},
		"properties": map[string]any{
			"title":       map[string]any{"type": "string", "description": "Short heading for the card."},
			"description": map[string]any{"type": "string", "description": "One short sentence of context under the heading."},
			"questions":   map[string]any{"type": "array", "minItems": 1, "maxItems": s.max(), "items": question},
		},
	}
}
