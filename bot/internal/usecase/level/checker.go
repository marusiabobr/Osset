package level

import (
	"context"
	"fmt"
	"strings"

	"lingw/internal/domain"
)

type Checker struct {
	lexicon domain.LexiconStore
}

func NewChecker(lexicon domain.LexiconStore) *Checker {
	return &Checker{lexicon: lexicon}
}

func (c *Checker) Check(ctx context.Context, exercise domain.Exercise, answer string) (bool, error) {
	switch exercise.Type {
	case domain.ExerciseTheory:
		if hasAcceptedData(exercise.Data) {
			return c.checkByRefs(ctx, exercise, answer)
		}
		return true, nil
	case domain.ExerciseVocab:
		if hasAcceptedData(exercise.Data) {
			return c.checkByRefs(ctx, exercise, answer)
		}
		return true, nil
	case domain.ExerciseChoice, domain.ExerciseFillBlank, domain.ExerciseTranslateOS, domain.ExerciseTranslateRU:
		return c.checkByRefs(ctx, exercise, answer)
	case domain.ExerciseMatch:
		if _, ok := exercise.Data["accepted_refs"]; ok {
			return c.checkByRefs(ctx, exercise, answer)
		}
		correctRaw, ok := exercise.Data["correct"].(string)
		if !ok {
			return false, fmt.Errorf("exercise match missing correct")
		}
		return normalize(correctRaw) == normalize(answer), nil
	default:
		return false, fmt.Errorf("unsupported exercise type: %s", exercise.Type)
	}
}

func (c *Checker) checkByRefs(ctx context.Context, exercise domain.Exercise, answer string) (bool, error) {
	switch exercise.Type {
	case domain.ExerciseTheory, domain.ExerciseVocab, domain.ExerciseChoice, domain.ExerciseFillBlank, domain.ExerciseTranslateOS, domain.ExerciseTranslateRU, domain.ExerciseMatch:
		literals := literalsFromData(exercise.Data, "accepted_literals")
		if len(literals) > 0 && containsNormalized(literals, answer) {
			return true, nil
		}
		refs, ok := exercise.Data["accepted_refs"].([]interface{})
		if (!ok || len(refs) == 0) && len(literals) > 0 {
			return false, nil
		}
		if !ok || len(refs) == 0 {
			return false, fmt.Errorf("exercise %s missing accepted_refs", exercise.Type)
		}
		converted := make([]string, 0, len(refs))
		for _, r := range refs {
			converted = append(converted, fmt.Sprintf("%v", r))
		}
		accepted, err := c.lexicon.AcceptedAnswers(ctx, converted)
		if err != nil {
			return false, err
		}
		return containsNormalized(accepted, answer), nil
	default:
		return false, fmt.Errorf("unsupported exercise type for refs: %s", exercise.Type)
	}
}

func literalsFromData(data map[string]interface{}, key string) []string {
	raw, ok := data[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, fmt.Sprintf("%v", v))
	}
	return out
}

func hasAcceptedData(data map[string]interface{}) bool {
	if refs, ok := data["accepted_refs"].([]interface{}); ok && len(refs) > 0 {
		return true
	}
	if literals, ok := data["accepted_literals"].([]interface{}); ok && len(literals) > 0 {
		return true
	}
	return false
}

func containsNormalized(accepted []string, answer string) bool {
	normAns := normalize(answer)
	for _, val := range accepted {
		if normalize(val) == normAns {
			return true
		}
	}
	return false
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, "ё", "е")
	return v
}
