package level

import (
	"testing"

	"lingw/internal/domain"
)

func theory(prompt string) domain.Exercise {
	return domain.Exercise{
		Type: domain.ExerciseTheory,
		Data: map[string]interface{}{"prompt": prompt},
	}
}

func choiceCase(prompt string) domain.Exercise {
	return domain.Exercise{
		Type: domain.ExerciseChoice,
		Data: map[string]interface{}{"prompt": prompt},
	}
}

func choiceForm(targetCase string) domain.Exercise {
	return domain.Exercise{
		Type: domain.ExerciseChoice,
		Data: map[string]interface{}{
			"target_case": targetCase,
			"prompt":      "выберите форму",
		},
	}
}

func TestIsShufflablePractice(t *testing.T) {
	t.Parallel()
	if isShufflablePractice(theory("Падеж Именительный")) {
		t.Error("theory should not shuffle")
	}
	if !isShufflablePractice(choiceCase("Какой падеж у формы «азы»?")) {
		t.Error("case drill should shuffle")
	}
	if !isShufflablePractice(choiceForm("Родительный")) {
		t.Error("choose_form should shuffle")
	}
	ex := domain.Exercise{
		Type: domain.ExerciseChoice,
		Data: map[string]interface{}{"match_case": true},
	}
	if !isShufflablePractice(ex) {
		t.Error("match_case segment should shuffle")
	}
}

func TestExerciseOrderPreservesTheoryBlocks(t *testing.T) {
	t.Parallel()
	exercises := []domain.Exercise{
		theory("rule 1"),
		theory("rule 2"),
		choiceCase("падеж 1"),
		choiceCase("падеж 2"),
		theory("rule 3"),
	}
	order := exerciseOrder(exercises, 42, "topic_01_level_01")
	if order[0] != 0 || order[1] != 1 || order[4] != 4 {
		t.Fatalf("theory indices moved: %v", order)
	}
	if order[2] == order[3] {
		t.Fatal("practice segment should contain both drills")
	}
}

func TestExerciseOrderStablePerUser(t *testing.T) {
	t.Parallel()
	exercises := []domain.Exercise{
		choiceCase("падеж A"),
		choiceCase("падеж B"),
		choiceCase("падеж C"),
	}
	a := exerciseOrder(exercises, 1, "lvl")
	b := exerciseOrder(exercises, 1, "lvl")
	c := exerciseOrder(exercises, 2, "lvl")
	if len(a) != 3 {
		t.Fatalf("order len: %d", len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("order not stable for same user: %v vs %v", a, b)
		}
	}
	same := true
	for i := range a {
		if a[i] != c[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different order for different users")
	}
}

func TestExerciseAtBounds(t *testing.T) {
	t.Parallel()
	exercises := []domain.Exercise{theory("only")}
	_, ok := exerciseAt(exercises, -1, 1, "x")
	if ok {
		t.Error("negative step should be out of bounds")
	}
	_, ok = exerciseAt(exercises, 1, 1, "x")
	if ok {
		t.Error("step beyond length should be out of bounds")
	}
	ex, ok := exerciseAt(exercises, 0, 1, "x")
	if !ok || ex.Data["prompt"] != "only" {
		t.Fatalf("expected first exercise, got ok=%v", ok)
	}
}
