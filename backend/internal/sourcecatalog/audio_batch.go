package sourcecatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"github.com/jackc/pgx/v5"
)

type AudioCoverage struct {
	Reviewed    int64 `json:"reviewed"`
	Pending     int64 `json:"pending"`
	Unsupported int64 `json:"unsupported"`
	Held        int64 `json:"held"`
	Retry       int64 `json:"retry"`
	Unavailable int64 `json:"unavailable"`
}

// Coverage counts usable current bindings, rather than historical review attempts.
func (s *Store) AudioCoverage(ctx context.Context) (AudioCoverage, error) {
	var c AudioCoverage
	err := s.Core.Pool.QueryRow(ctx, `WITH playable AS (
 SELECT DISTINCT r.source_item_id,r.source_hash FROM catalog_representations r
 JOIN representation_transport_reviews t ON t.item_id=r.item_id AND t.representation_id=r.id AND t.collection_id=$1
 JOIN catalog_memberships cm ON cm.item_id=r.item_id AND cm.collection_id=$1 AND cm.listed
 JOIN catalog_representation_memberships rm ON rm.item_id=r.item_id AND rm.representation_id=r.id AND rm.collection_id=$1 AND rm.listed
 WHERE r.media='audio' AND t.review->>'snapshot_sha256'=r.source_hash AND t.review->>'playback'='true'),
 coverage AS (
 SELECT ci.unavailable,a.state,p.source_item_id IS NOT NULL AS playable
 FROM source_catalog_items ci LEFT JOIN source_catalog_audio_reviews a ON a.collection_id=ci.collection_id AND a.item_id=ci.identifier AND a.source_hash=ci.source_hash
 LEFT JOIN playable p ON p.source_item_id=ci.identifier AND p.source_hash=ci.source_hash
 WHERE ci.collection_id=$1 AND NOT ci.excluded)
 SELECT count(*) FILTER(WHERE NOT unavailable AND playable),
 count(*) FILTER(WHERE NOT unavailable AND NOT playable AND state IS NULL),
 count(*) FILTER(WHERE NOT unavailable AND NOT playable AND state='unsupported'),
 count(*) FILTER(WHERE NOT unavailable AND NOT playable AND state IN ('held','reviewed')),
 count(*) FILTER(WHERE NOT unavailable AND NOT playable AND state='retry'),
 count(*) FILTER(WHERE unavailable) FROM coverage`, s.Collection).Scan(&c.Reviewed, &c.Pending, &c.Unsupported, &c.Held, &c.Retry, &c.Unavailable)
	return c, err
}

func (s *Store) saveAudioAttempt(ctx context.Context, id, hash, state, message string, review []byte) error {
	_, err := s.Core.Pool.Exec(ctx, `INSERT INTO source_catalog_audio_reviews(collection_id,item_id,source_hash,state,error,review,next_attempt)
 VALUES($1,$2,$3,$4,$5,$6,now()+interval '1 hour') ON CONFLICT(collection_id,item_id) DO UPDATE SET source_hash=excluded.source_hash,state=excluded.state,error=excluded.error,review=excluded.review,attempts=source_catalog_audio_reviews.attempts+1,next_attempt=excluded.next_attempt,updated_at=now()`, s.Collection, id, hash, state, message, review)
	return err
}

// ReviewAudioBatch is an explicit, bounded private playback approval. It does
// not fetch metadata or media, and never replaces a prior independent decision.
func (s *Store) ReviewAudioBatch(ctx context.Context, limit int, notes string) (AudioCoverage, error) {
	if limit < 1 || limit > 1000 || strings.TrimSpace(notes) == "" || len(notes) > 4000 {
		return AudioCoverage{}, fmt.Errorf("bounded batch size and explicit playback approval notes required")
	}
	collection, err := s.Core.GetCollection(ctx, s.Collection)
	if err != nil {
		return AudioCoverage{}, err
	}
	if collection.Policy != domain.Stream {
		return AudioCoverage{}, fmt.Errorf("batch playback approval requires a source-streaming collection")
	}
	conn, err := s.Core.Pool.Acquire(ctx)
	if err != nil {
		return AudioCoverage{}, err
	}
	defer conn.Release()
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended(current_schema() || ':source-audio:' || $1,7920))`, s.Collection).Scan(&locked); err != nil {
		return AudioCoverage{}, err
	}
	if !locked {
		return AudioCoverage{}, fmt.Errorf("source audio reviewer is already running")
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(hashtextextended(current_schema() || ':source-audio:' || $1,7920))`, s.Collection)
	rows, err := s.Core.Pool.Query(ctx, `SELECT ci.identifier FROM source_catalog_items ci LEFT JOIN source_catalog_audio_reviews a ON a.collection_id=ci.collection_id AND a.item_id=ci.identifier
 WHERE ci.collection_id=$1 AND NOT ci.excluded AND NOT ci.unavailable AND (a.item_id IS NULL OR a.source_hash<>ci.source_hash OR (a.state='retry' AND a.next_attempt<=now()))
 ORDER BY ci.imported_at,ci.identifier LIMIT $2`, s.Collection, limit)
	if err != nil {
		return AudioCoverage{}, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return AudioCoverage{}, err
	}
	for _, id := range ids {
		detail, err := s.Detail(ctx, id)
		if err != nil {
			return AudioCoverage{}, err
		}
		hash := detail["snapshot_sha256"].(string)
		if detail["unavailable"] == true {
			continue
		}
		var bindings []json.RawMessage
		if err = json.Unmarshal(detail["playback_bindings"].(json.RawMessage), &bindings); err != nil {
			return AudioCoverage{}, err
		}
		if len(bindings) > 0 {
			if err = s.saveAudioAttempt(ctx, id, hash, "reviewed", "", nil); err != nil {
				return AudioCoverage{}, err
			}
			continue
		}
		var count int
		var managed bool
		err = s.Core.Pool.QueryRow(ctx, `SELECT count(*),COALESCE(bool_and(a.review IS NOT NULL AND a.review=t.review),false)
 FROM catalog_representations r JOIN representation_transport_reviews t ON t.item_id=r.item_id AND t.representation_id=r.id AND t.collection_id=$1
 LEFT JOIN source_catalog_audio_reviews a ON a.collection_id=$1 AND a.item_id=$2
 WHERE r.source_item_id=$2 AND r.media='audio'`, s.Collection, id).Scan(&count, &managed)
		if err != nil {
			return AudioCoverage{}, err
		}
		state, message := "reviewed", ""
		var review []byte
		if count > 0 && !managed {
			state, message = "held", "existing independent playback decision requires operator review"
		} else if err = s.ReviewAudio(ctx, id, notes); err != nil {
			if ctx.Err() != nil {
				return AudioCoverage{}, ctx.Err()
			}
			state, message = "retry", err.Error()
			if errors.Is(err, ErrUnsupportedAudio) {
				state = "unsupported"
			}
		} else {
			itemID := "audio-" + domain.Digest([]byte(s.Collection + "\x00" + id))[:24]
			if err = s.Core.Pool.QueryRow(ctx, `SELECT review FROM representation_transport_reviews WHERE item_id=$1 AND representation_id='audio' AND collection_id=$2`, itemID, s.Collection).Scan(&review); err != nil {
				return AudioCoverage{}, err
			}
		}
		if err = s.saveAudioAttempt(ctx, id, hash, state, message, review); err != nil {
			return AudioCoverage{}, err
		}
	}
	return s.AudioCoverage(ctx)
}
