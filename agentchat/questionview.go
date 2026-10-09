package agentchat

import (
	"slices"
	"strconv"
	"strings"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

// QuestionCardView is what QuestionForm renders: the card and the session
// its answers belong to. Hidden fields (a CSRF token) are rendered as hidden
// inputs in key order, so the card posts them without JavaScript; the card's
// own session_id, answer_questions and question_token are reserved and skipped.
type QuestionCardView struct {
	SessionID string
	Card      QuestionCard
	Hidden    map[string]string
}

func (v QuestionCardView) hiddenKeys() []string {
	return slices.DeleteFunc(sortedKeys(v.Hidden), func(k string) bool {
		return k == "session_id" || k == "answer_questions" || k == "question_token"
	})
}

func (q Question) isChoice() bool { return isChoice(q.Type) }

func (q Question) hasPrefill() bool { return q.Value != "" || len(q.Values) > 0 }

func (q Question) chips() []c.ChipOption {
	out := make([]c.ChipOption, 0, len(q.Options))
	for _, o := range q.Options {
		selected := q.Value == o
		for _, v := range q.Values {
			selected = selected || v == o
		}
		out = append(out, c.ChipOption{Value: o, Label: o, Selected: selected})
	}
	return out
}

// autoSubmit: a card that is one unanswered buttons question sends as soon as a chip is picked.
func (cd QuestionCard) autoSubmit() bool {
	return len(cd.Questions) == 1 && cd.Questions[0].Type == TypeButtons && !cd.Questions[0].hasPrefill()
}

func (cd QuestionCard) heading() string {
	return or(strings.TrimSpace(cd.Title), "A few quick questions")
}

func (cd QuestionCard) anyPrefill() bool {
	for _, q := range cd.Questions {
		if q.hasPrefill() {
			return true
		}
	}
	return false
}

// fieldID is unique per card: the token keeps two cards on a page apart.
func (cd QuestionCard) fieldID(q Question) string {
	token := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cd.Token)
	return "q-" + token[:min(len(token), 8)] + "-" + q.Key
}

// groupDigits formats "11000" as "11,000" for a number input; the runtime
// regroups on blur and strips the commas on send.
func groupDigits(v string) string {
	v = strings.TrimSpace(v)
	n, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", ""), 64)
	if err != nil {
		return v
	}
	whole, frac, _ := strings.Cut(strconv.FormatFloat(n, 'f', -1, 64), ".")
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 && r != '-' && whole[i-1] != '-' {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if frac != "" {
		b.WriteString("." + frac)
	}
	return b.String()
}

func (q Question) fieldOpts() c.FieldOpts {
	return c.FieldOpts{Help: q.Help, Required: q.Required, Suggested: q.Value != ""}
}

func describedBy(cd QuestionCard, q Question) string {
	return c.FieldDescribedBy(cd.fieldID(q), q.fieldOpts())
}

func itoa(n int) string { return strconv.Itoa(n) }
