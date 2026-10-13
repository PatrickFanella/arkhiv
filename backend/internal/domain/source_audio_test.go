package domain

import "testing"

func TestNestedAudioPaths(t *testing.T) {
	for _, name := range []string{"disc 1/01  Song.mp3", "disc/percent%25 #?.mp3"} {
		if !SourceAudioFile(name) {
			t.Fatal(name)
		}
	}
	for _, name := range []string{"", "/song.mp3", "disc//song.mp3", "../song.mp3", "disc/../song.mp3", "disc/./song.mp3", "disc\\song.mp3", "disc/\nsong.mp3"} {
		if SourceAudioFile(name) {
			t.Fatal("unsafe source path", name)
		}
	}
	if path := SourceAudioPath("disc 1/percent%25 #?.mp3"); path != "disc%201/percent%2525%20%23%3F.mp3" {
		t.Fatal(path)
	}
}
