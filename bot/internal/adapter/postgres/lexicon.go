package postgres

import (
	"context"

	"lingw/internal/domain"
)

type LexiconStore struct{}

func NewLexiconStore() *LexiconStore { return &LexiconStore{} }

func (s *LexiconStore) Resolve(context.Context, string) (domain.LexemeDisplay, error) {
	return domain.LexemeDisplay{}, domain.ErrNotImplemented
}

func (s *LexiconStore) ResolveForm(context.Context, string) (domain.WordFormDisplay, error) {
	return domain.WordFormDisplay{}, domain.ErrNotImplemented
}

func (s *LexiconStore) AcceptedAnswers(context.Context, []string) ([]string, error) {
	return nil, domain.ErrNotImplemented
}
