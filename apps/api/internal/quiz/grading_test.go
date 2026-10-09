package quiz

import "testing"

func tr(v bool) *bool { return &v }

func gradingQuestions() []Question {
	return []Question{
		{ID: "c1", Kind: "choice", Points: "2.00", Choices: []Choice{{"A", tr(true)}, {"B", tr(false)}, {"C", tr(false)}}},
		{ID: "c2", Kind: "choice", Points: "1.50", Choices: []Choice{{"A", tr(true)}, {"B", tr(true)}, {"C", tr(false)}}},
		{ID: "b1", Kind: "blank", Points: "3.25", Choices: []Choice{}, Answers: &[]string{"Paris", "  New   York "}},
	}
}

func TestAutoGrade(t *testing.T) {
	type want struct {
		points string
		ok     bool
	}
	cases := []struct {
		name    string
		answers map[string]Answer
		c1, c2  want
		b1      want
		total   string
	}{
		{"all right", map[string]Answer{"c1": {Choices: []int{0}}, "c2": {Choices: []int{1, 0}}, "b1": {Text: "  PARIS "}},
			want{"2.00", true}, want{"1.50", true}, want{"3.25", true}, "6.75"},
		{"wrong choice", map[string]Answer{"c1": {Choices: []int{1}}},
			want{"0.00", false}, want{"0.00", false}, want{"0.00", false}, "0.00"},
		{"duplicates are a set", map[string]Answer{"c1": {Choices: []int{0, 0}}},
			want{"2.00", true}, want{"0.00", false}, want{"0.00", false}, "2.00"},
		{"subset fails", map[string]Answer{"c2": {Choices: []int{0}}},
			want{"0.00", false}, want{"0.00", false}, want{"0.00", false}, "0.00"},
		{"superset fails", map[string]Answer{"c2": {Choices: []int{0, 1, 2}}},
			want{"0.00", false}, want{"0.00", false}, want{"0.00", false}, "0.00"},
		{"second accepted answer, inner whitespace and tab", map[string]Answer{"b1": {Text: "new\t york"}},
			want{"0.00", false}, want{"0.00", false}, want{"3.25", true}, "3.25"},
		{"misspelling", map[string]Answer{"b1": {Text: "Pariss"}},
			want{"0.00", false}, want{"0.00", false}, want{"0.00", false}, "0.00"},
		{"empty text, empty choices", map[string]Answer{"b1": {Text: ""}, "c1": {Choices: []int{}}},
			want{"0.00", false}, want{"0.00", false}, want{"0.00", false}, "0.00"},
		{"unknown key ignored", map[string]Answer{"x": {Text: "Paris"}},
			want{"0.00", false}, want{"0.00", false}, want{"0.00", false}, "0.00"},
		{"out of range index never matches", map[string]Answer{"c1": {Choices: []int{0, 7}}},
			want{"0.00", false}, want{"0.00", false}, want{"0.00", false}, "0.00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marks, total := autoGrade(gradingQuestions(), c.answers)
			if total != c.total || len(marks) != 3 {
				t.Fatalf("total %q marks %d", total, len(marks))
			}
			for id, w := range map[string]want{"c1": c.c1, "c2": c.c2, "b1": c.b1} {
				m, ok := marks[id]
				if !ok || string(m.Points) != w.points || m.Correct != w.ok || m.Feedback != "" {
					t.Errorf("%s: %+v want %+v", id, m, w)
				}
			}
		})
	}
	t.Run("exact decimals", func(t *testing.T) {
		qs := []Question{}
		ans := map[string]Answer{}
		for _, id := range []string{"x", "y", "z"} {
			qs = append(qs, Question{ID: id, Kind: "blank", Points: "0.10", Answers: &[]string{"a"}})
			ans[id] = Answer{Text: "a"}
		}
		if _, total := autoGrade(qs, ans); total != "0.30" {
			t.Fatalf("total %q", total)
		}
	})
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{"  New \t York\n": "new york", "PARIS": "paris", "": "", " \t ": ""} {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q want %q", in, got, want)
		}
	}
}

func TestRegradeMarks(t *testing.T) {
	yes, no := true, false
	qs := []Question{
		{ID: "a", Kind: "choice", Points: "0.10", Choices: []Choice{{Text: "x", Correct: &yes}, {Text: "y", Correct: &no}}},
		{ID: "b", Kind: "blank", Points: "0.20", Answers: &[]string{"p"}},
		{ID: "c", Kind: "blank", Points: "0.30", Answers: &[]string{"p"}},
		{ID: "d", Kind: "blank", Points: "0.40", Answers: &[]string{"p"}},
	}
	ans := map[string]Answer{"a": {Choices: []int{0}}, "b": {Text: "p"}, "c": {Text: "p"}, "d": {Text: "q"}}
	old := map[string]Mark{
		"a": {Points: "0.00", Feedback: "fa"},
		"b": {Points: "0.05", Feedback: "fb", Manual: true},
		"c": {Points: "0.00", Feedback: "fc"},
	}
	got, total := regradeMarks(qs, ans, old)
	if got["a"].Points != "0.10" || !got["a"].Correct || got["a"].Feedback != "fa" || got["a"].Manual {
		t.Errorf("a %+v", got["a"])
	}
	if got["b"] != old["b"] {
		t.Errorf("b %+v", got["b"])
	}
	if got["c"].Points != "0.30" || got["c"].Feedback != "fc" {
		t.Errorf("c %+v", got["c"])
	}
	if got["d"].Points != "0.00" || got["d"].Feedback != "" {
		t.Errorf("d %+v", got["d"])
	}
	if total != "0.45" {
		t.Errorf("total %s", total)
	}
	// lowered below the reviewed value: capped, correct, still manual; raised: untouched
	qs[1].Points = "0.04"
	got, total = regradeMarks(qs, ans, old)
	if b := got["b"]; b.Points != "0.04" || !b.Correct || !b.Manual || b.Feedback != "fb" || total != "0.44" {
		t.Errorf("capped %+v total %s", b, total)
	}
	qs[1].Points = "9.00"
	if got, _ = regradeMarks(qs, ans, old); got["b"] != old["b"] {
		t.Errorf("raised %+v", got["b"])
	}
}
