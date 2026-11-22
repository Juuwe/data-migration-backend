package service

import (
	"context"
	"testing"
)

type feedRepositoryStub struct {
	sets  [][]int64
	calls int
}

func (s *feedRepositoryStub) FindRandomPublishedIDs(context.Context, int) ([]int64, error) {
	if s.calls >= len(s.sets) {
		return s.sets[len(s.sets)-1], nil
	}
	ids := s.sets[s.calls]
	s.calls++
	return ids, nil
}

type feedStoreStub struct {
	ids []int64
}

func (s *feedStoreStub) GetIDs(context.Context, string) ([]int64, error) {
	return append([]int64(nil), s.ids...), nil
}

func (s *feedStoreStub) SetIDs(_ context.Context, _ string, ids []int64) error {
	s.ids = append([]int64(nil), ids...)
	return nil
}

func TestFeedServiceFirstAndNext(t *testing.T) {
	repo := &feedRepositoryStub{sets: [][]int64{{1, 2, 3}, {4, 5, 6}}}
	store := &feedStoreStub{}
	svc := NewFeedService(repo, store, 3)

	first, err := svc.First(context.Background(), "session")
	if err != nil {
		t.Fatalf("First() error = %v", err)
	}
	if first.MethodID != 1 || first.Position != 1 || first.Size != 3 {
		t.Fatalf("First() = %+v", first)
	}

	next, err := svc.Next(context.Background(), "session", 1)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if next.MethodID != 2 || next.Position != 2 {
		t.Fatalf("Next() = %+v", next)
	}

	regenerated, err := svc.Next(context.Background(), "session", 3)
	if err != nil {
		t.Fatalf("Next() after end error = %v", err)
	}
	if regenerated.MethodID != 4 || regenerated.Position != 1 {
		t.Fatalf("Next() after end = %+v", regenerated)
	}
	if repo.calls != 2 {
		t.Fatalf("generation calls = %d, want 2", repo.calls)
	}
}

func TestFeedServiceReusesStoredFeed(t *testing.T) {
	repo := &feedRepositoryStub{sets: [][]int64{{7, 8}}}
	store := &feedStoreStub{ids: []int64{1, 2, 3}}
	svc := NewFeedService(repo, store, 3)

	first, err := svc.First(context.Background(), "session")
	if err != nil {
		t.Fatalf("First() error = %v", err)
	}
	if first.MethodID != 1 || first.Size != 3 {
		t.Fatalf("First() = %+v", first)
	}
	if repo.calls != 0 {
		t.Fatalf("generation calls = %d, want 0", repo.calls)
	}
}
