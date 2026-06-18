package level

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"lingw/internal/domain"
)

func exerciseAt(exercises []domain.Exercise, step int, userID int64, levelSlug string) (domain.Exercise, bool) {
	order := exerciseOrder(exercises, userID, levelSlug)
	if step < 0 || step >= len(order) {
		return domain.Exercise{}, false
	}
	return exercises[order[step]], true
}

func exerciseOrder(exercises []domain.Exercise, userID int64, levelSlug string) []int {
	order := make([]int, len(exercises))
	for i := range order {
		order[i] = i
	}
	i := 0
	for i < len(exercises) {
		if !isShufflablePractice(exercises[i]) {
			i++
			continue
		}
		start := i
		for i < len(exercises) && isShufflablePractice(exercises[i]) {
			i++
		}
		segment := append([]int(nil), order[start:i]...)
		shuffleSegment(segment, userID, levelSlug, start)
		copy(order[start:i], segment)
	}
	return order
}

// isShufflablePractice marks case drills (choice exercises), not theory cards about cases.
func isShufflablePractice(ex domain.Exercise) bool {
	if ex.Type != domain.ExerciseChoice {
		return false
	}
	if _, ok := ex.Data["match_case"].(bool); ok {
		return true
	}
	if _, ok := ex.Data["target_case"].(string); ok {
		return true
	}
	if prompt, ok := ex.Data["prompt"].(string); ok {
		lower := strings.ToLower(prompt)
		return strings.Contains(lower, "падеж") || strings.Contains(lower, "парадигма")
	}
	return false
}

func shuffleSegment(segment []int, userID int64, levelSlug string, segmentStart int) {
	seed := fmt.Sprintf("%d:%s:%d", userID, levelSlug, segmentStart)
	sort.Slice(segment, func(a, b int) bool {
		ha := md5.Sum([]byte(fmt.Sprintf("%s:%d", seed, segment[a])))
		hb := md5.Sum([]byte(fmt.Sprintf("%s:%d", seed, segment[b])))
		return bytes.Compare(ha[:], hb[:]) < 0
	})
}
