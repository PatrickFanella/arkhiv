package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/community"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (s *Server) communityAvailable(ctx context.Context, collection, id string) bool {
	if !s.allowsCollection(collection) {
		return false
	}
	entry, err := s.DB.Entry(ctx, id)
	if err != nil || entry.Episode.CollectionID != collection || !entry.Decision.Published || entry.Unavailable {
		return false
	}
	_, err = domain.Resolve(entry, false)
	return err == nil
}
func (s *Server) communityCurrent(ctx context.Context, collection string) (*community.Program, error) {
	if !s.allowsCollection(collection) {
		return nil, fmt.Errorf("collection unavailable")
	}
	schedule, err := s.DB.Schedule(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	within, err := s.DB.ScheduleWithinScope(ctx, schedule, domain.CollectionScope{collection})
	if err != nil || !within {
		return nil, err
	}
	now := time.Now()
	if s.Clock != nil {
		now = s.Clock()
	}
	entries, err := s.DB.CurrentEntries(ctx, schedule, now)
	if err != nil {
		return nil, err
	}
	state := schedule.At(now, entries)
	if state.State != "playing" || state.Program == nil || state.Episode == nil || state.Episode.CollectionID != collection {
		return nil, nil
	}
	return &community.Program{Revision: state.Revision, Start: state.Program.Start, Episode: state.Program.EpisodeID, OriginalDate: state.Episode.Date}, nil
}
