package community

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/testsupport"
)

func TestMixedReplayRoomsAndRecoveryIntegration(t *testing.T) {
	pool, owner, _ := testsupport.Personal(t)
	ctx := context.Background()
	public, key, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Now().UTC()
	video := Program{Revision: "video-revision", Start: now.Add(-time.Minute), Episode: "episode", OriginalDate: "1977-05-08"}
	audio := Program{Media: "audio", Revision: "audio-revision", Start: now.Add(-time.Minute), Item: "recording", Representation: "continuous", OriginalDate: "1969-06-21"}
	s := &Server{Pool: pool, Trust: Trust{Key: public, Issuer: "consumer", Deployment: "brand"}, Clock: func() time.Time { return now }, Allowed: func(c string) bool { return c == "mixed" }, ChatRead: true, ChatPost: true, AudioChatRead: true, AudioChatPost: true, Moderators: map[string]bool{Hash("provider\x00video-mod"): true}, AudioModerators: map[string]bool{Hash("provider\x00audio-mod"): true}, RecoveryDirectory: t.TempDir()}
	if err := os.Chmod(s.RecoveryDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	s.CurrentMedia = func(_ context.Context, c, media string) (*Program, error) {
		if c != "mixed" {
			return nil, nil
		}
		if media == "audio" {
			return &audio, nil
		}
		return &video, nil
	}
	mux := http.NewServeMux()
	s.Register(mux)
	tokens := map[string]string{}
	for _, subject := range []string{"viewer", "video-mod", "audio-mod"} {
		hash := Hash(subject)
		_, err := owner.Exec(ctx, `INSERT INTO personal_sessions VALUES($1,'brand','provider',$2,$3,$4)`, hash, subject, now, now.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		token, err := Sign(key, "consumer", "brand", "provider", subject, hash, []string{"personal", "chat", "moderate"}, now)
		if err != nil {
			t.Fatal(err)
		}
		tokens[subject] = token
	}
	request := func(subject, method, path string, body any, status int) *httptest.ResponseRecorder {
		t.Helper()
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
		if subject != "" {
			r.Header.Set("Authorization", "Bearer "+tokens[subject])
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s got %d: %s", method, path, w.Code, w.Body.String())
		}
		return w
	}
	post := func(p Program, k string) Submission {
		return Submission{Program: p, Key: k, Body: "<b>literal discussion</b>", Mode: "channel"}
	}
	var a, v Message
	if err := json.Unmarshal(request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", post(audio, "shared-key"), 201).Body.Bytes(), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages", post(video, "shared-key"), 201).Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", post(audio, "shared-key"), 200)
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages", post(video, "shared-key"), 200)
	for _, media := range []string{"video", "audio"} {
		result := request("", "GET", "/api/chat/v1/rooms/mixed/messages?media="+media, nil, 200).Body.String()
		want, other := v.ID, a.ID
		if media == "audio" {
			want, other = a.ID, v.ID
		}
		if !strings.Contains(result, want) || strings.Contains(result, other) {
			t.Fatal("room leaked foreign media")
		}
	}
	request("", "GET", "/api/chat/v1/rooms/mixed/messages?media=audio&after=1", nil, 400)
	request("", "GET", "/api/chat/v1/rooms/mixed/messages?after=audio:1", nil, 400)
	request("", "GET", fmt.Sprintf("/api/chat/v1/rooms/mixed/messages?media=audio&after=audio:%d", a.Seq), nil, 200)
	request("", "GET", "/api/chat/v1/rooms/mixed?media=print", nil, 400)
	for _, bad := range []Submission{post(video, "wrong-media"), post(Program{Media: "audio", Revision: audio.Revision, Start: audio.Start, Item: "foreign", Representation: audio.Representation}, "foreign-item"), post(Program{Media: "audio", Revision: audio.Revision, Start: audio.Start, Item: audio.Item, Representation: "foreign"}, "foreign-representation")} {
		request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", bad, 409)
	}
	bad := post(audio, "paused")
	bad.Paused = true
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", bad, 409)
	bad = post(audio, "ondemand")
	bad.Mode = "ondemand"
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", bad, 409)
	bad = post(audio, "stale")
	bad.Revision = "old"
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", bad, 409)
	request("viewer", "POST", "/api/chat/v1/rooms/foreign/messages?media=audio", post(audio, "foreign-room"), 404)
	for i := 0; i < 4; i++ {
		request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", post(audio, fmt.Sprint(i)), 201)
	}
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", post(audio, "limited"), 429)
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages", post(video, "video-independent-rate"), 201)
	request("video-mod", "GET", "/api/chat/v1/moderation?media=audio", nil, 403)
	request("audio-mod", "GET", "/api/chat/v1/moderation", nil, 403)
	request("audio-mod", "POST", "/api/chat/v1/moderation?media=audio", map[string]any{"action": "hide", "target": v.ID, "reason": "wrong scope"}, 404)
	request("audio-mod", "POST", "/api/chat/v1/moderation?media=audio", map[string]any{"action": "hide", "target": a.ID, "reason": "reviewed"}, 200)
	request("audio-mod", "POST", "/api/chat/v1/moderation?media=audio", map[string]any{"action": "ban", "target": a.UserID, "reason": "reviewed", "hours": 24}, 200)
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages", post(video, "audio-ban-not-video"), 201)
	checkpoint, err := RecoveryCheckpoint(ctx, pool, "brand", now)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err = MergeRecoveryJournal(checkpoint, s.RecoveryDirectory, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(checkpoint.Restrictions) != 1 || checkpoint.Restrictions[0].Media != "audio" {
		t.Fatal("restriction lost media")
	}
	restored, restoreOwner, _ := testsupport.Personal(t)
	if err = ReapplyRecovery(ctx, restored, checkpoint, "brand", now); err != nil {
		t.Fatal(err)
	}
	var media string
	if err = restoreOwner.QueryRow(ctx, `SELECT media FROM chat_restrictions WHERE user_id=$1`, a.UserID).Scan(&media); err != nil || media != "audio" {
		t.Fatal("restored restriction leaked scope", err)
	}
	s.AudioChatPost = false
	request("viewer", "POST", "/api/chat/v1/rooms/mixed/messages?media=audio", post(audio, "post-disabled"), 403)
	s.AudioChatRead = false
	request("", "GET", "/api/chat/v1/rooms/mixed/messages?media=audio", nil, 403)
	request("", "GET", "/api/chat/v1/rooms/mixed/messages", nil, 200)
}
