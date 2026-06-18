package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"

	"lingw/internal/domain"
	"lingw/seeds"
)

type LexiconStore struct {
	data map[string]domain.LexemeDisplay
}

func NewLexiconStore() (*LexiconStore, error) {
	raw, err := fs.ReadFile(seeds.Files, "lexicon_stub.json")
	if err != nil {
		return nil, fmt.Errorf("read lexicon_stub.json: %w", err)
	}
	var decoded map[string]domain.LexemeDisplay
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode lexicon_stub.json: %w", err)
	}
	return &LexiconStore{data: decoded}, nil
}

func (s *LexiconStore) Resolve(_ context.Context, ref string) (domain.LexemeDisplay, error) {
	val, ok := s.data[ref]
	if !ok {
		return domain.LexemeDisplay{}, domain.ErrNotFound
	}
	return val, nil
}

func (s *LexiconStore) ResolveForm(ctx context.Context, ref string) (domain.WordFormDisplay, error) {
	val, err := s.Resolve(ctx, ref)
	if err != nil {
		return domain.WordFormDisplay{}, err
	}
	return domain.WordFormDisplay{OS: val.OS, RU: val.RU}, nil
}

func (s *LexiconStore) AcceptedAnswers(ctx context.Context, refs []string) ([]string, error) {
	out := make([]string, 0, len(refs)*2)
	for _, ref := range refs {
		val, err := s.Resolve(ctx, ref)
		if err != nil {
			return nil, err
		}
		if val.OS != "" {
			out = append(out, val.OS)
		}
		if val.RU != "" {
			out = append(out, val.RU)
		}
	}
	return out, nil
}
