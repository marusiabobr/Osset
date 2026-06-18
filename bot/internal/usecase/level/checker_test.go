package level

import (
	"context"
	"testing"

	"lingw/internal/domain"
	"lingw/internal/testutil"
)

func TestNormalize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"  Ёлка ", "елка"},
		{"ДАЛЕЕ", "далее"},
		{"район", "район"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := normalize(tc.in); got != tc.want {
			t.Errorf("normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestContainsNormalized(t *testing.T) {
	t.Parallel()
	accepted := []string{"Родительный", "  дательный ", "Ном"}
	if !containsNormalized(accepted, "родительный") {
		t.Error("expected match for родительный")
	}
	if !containsNormalized(accepted, "Дательный") {
		t.Error("expected case-insensitive match")
	}
	if containsNormalized(accepted, "Именительный") {
		t.Error("unexpected match")
	}
}

func TestCheckerAcceptedLiterals(t *testing.T) {
	t.Parallel()
	c := NewChecker(testutil.StubLexicon{})
	ex := domain.Exercise{
		Type: domain.ExerciseChoice,
		Data: map[string]interface{}{
			"accepted_literals": []interface{}{"Родительный", "родительный"},
		},
	}
	ok, err := c.Check(context.Background(), ex, "  РОДИТЕЛЬНЫЙ ")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !ok {
		t.Fatal("expected accepted literal to pass")
	}
}

func TestCheckerTheoryWithoutAcceptedData(t *testing.T) {
	t.Parallel()
	c := NewChecker(testutil.StubLexicon{})
	ex := domain.Exercise{Type: domain.ExerciseTheory, Data: map[string]interface{}{}}
	ok, err := c.Check(context.Background(), ex, "anything")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !ok {
		t.Fatal("theory without accepted data should auto-pass")
	}
}

func TestCheckerVocabDalее(t *testing.T) {
	t.Parallel()
	c := NewChecker(testutil.StubLexicon{})
	ex := domain.Exercise{
		Type: domain.ExerciseVocab,
		Data: map[string]interface{}{
			"accepted_literals": []interface{}{"далее"},
		},
	}
	ok, err := c.Check(context.Background(), ex, "Далее")
	if err != nil || !ok {
		t.Fatalf("vocab далее: ok=%v err=%v", ok, err)
	}
}
