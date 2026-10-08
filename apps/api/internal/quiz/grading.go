package quiz

import (
	"encoding/json"
	"math/big"
	"strings"
)

// Choice is one option of a multiple-choice question; Correct is nil when hidden from a student.
type Choice struct {
	Text    string `json:"text"`
	Correct *bool  `json:"correct,omitempty"`
}

// Question is one question of a quiz. Multiple is derived (more than one correct choice), never stored,
// and does not reveal which choices are correct.
type Question struct {
	ID       string      `json:"id"`
	Kind     string      `json:"kind"` // choice | blank
	Prompt   string      `json:"prompt"`
	Points   json.Number `json:"points"`
	Position int         `json:"position"`
	Multiple bool        `json:"multiple"`
	Choices  []Choice    `json:"choices"`           // [] for blank
	Answers  *[]string   `json:"answers,omitempty"` // nil = hidden; &[]string{} for choice
}

// Answer is a student's answer to one question.
type Answer struct {
	Choices []int  `json:"choices,omitempty"`
	Text    string `json:"text,omitempty"`
}

// Mark is the grade of one question.
type Mark struct {
	Points   json.Number `json:"points"`
	Correct  bool        `json:"correct"`
	Feedback string      `json:"feedback"`
}

func normalize(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }

// autoGrade marks every question of qs (one Mark per question id, unanswered = 0, correct false) and
// returns the total as a 2-decimal string. Pure: no clock, no database.
// ponytail: no Unicode normalisation (NFC), so "é" typed two ways differs; add golang.org/x/text when a course needs it (not allowed today, D1).
func autoGrade(qs []Question, answers map[string]Answer) (marks map[string]Mark, total string) {
	marks = make(map[string]Mark, len(qs))
	sum := new(big.Rat)
	for _, q := range qs {
		pts, _ := new(big.Rat).SetString(string(q.Points))
		ok := isRight(q, answers[q.ID])
		got := new(big.Rat)
		if ok {
			got = pts
			sum.Add(sum, pts)
		}
		marks[q.ID] = Mark{Points: json.Number(got.FloatString(2)), Correct: ok}
	}
	return marks, sum.FloatString(2)
}

func isRight(q Question, a Answer) bool {
	if q.Kind == "choice" {
		chosen := map[int]bool{}
		for _, i := range a.Choices {
			chosen[i] = true
		}
		if len(chosen) == 0 {
			return false
		}
		for i, c := range q.Choices {
			if chosen[i] != (c.Correct != nil && *c.Correct) {
				return false
			}
			delete(chosen, i)
		}
		return len(chosen) == 0 // indexes beyond the choices never match
	}
	got := normalize(a.Text)
	if got == "" || q.Answers == nil {
		return false
	}
	for _, want := range *q.Answers {
		if normalize(want) == got {
			return true
		}
	}
	return false
}
