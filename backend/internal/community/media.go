package community

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func normalizedMedia(media string) string {
	if media == "" {
		return "video"
	}
	return media
}
func validMedia(media string) bool { return media == "video" || media == "audio" }
func requestMedia(w http.ResponseWriter, r *http.Request) (string, bool) {
	media := normalizedMedia(r.URL.Query().Get("media"))
	if !validMedia(media) {
		fail(w, 400, "Use a video or audio room.")
		return "", false
	}
	return media, true
}
func (s *Server) current(ctx context.Context, collection, media string) (*Program, error) {
	var p *Program
	var err error
	if s.CurrentMedia != nil {
		p, err = s.CurrentMedia(ctx, collection, media)
	} else if media == "video" && s.Current != nil {
		p, err = s.Current(ctx, collection)
	}
	if p != nil {
		if normalizedMedia(p.Media) != media || media == "audio" && (!validName(p.Item) || !validName(p.Representation)) {
			return nil, nil
		}
		copy := *p
		copy.Media = media
		p = &copy
	}
	return p, err
}
func (s *Server) readable(media string) bool {
	if media == "audio" {
		return s.AudioChatRead
	}
	return s.ChatRead
}
func (s *Server) postable(media string) bool {
	if media == "audio" {
		return s.AudioChatRead && s.AudioChatPost
	}
	return s.ChatRead && s.ChatPost
}
func (s *Server) moderator(claims *Claims, media string) bool {
	roster := s.Moderators
	if media == "audio" {
		roster = s.AudioModerators
	}
	return roster[Hash(claims.Provider+"\x00"+claims.Subject)]
}

// Audio cursors cannot be reused in the legacy numeric video cursor lane.
func roomCursor(value, media string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	if media == "audio" {
		if !strings.HasPrefix(value, "audio:") {
			return 0, fmt.Errorf("audio cursor required")
		}
		value = strings.TrimPrefix(value, "audio:")
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if n < 0 {
		err = fmt.Errorf("invalid cursor")
	}
	return n, err
}
func encodeRoomCursor(value int64, media string) any {
	if media == "audio" {
		return fmt.Sprintf("audio:%d", value)
	}
	return value
}
