package activity

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func song() *Track {
	return &Track{
		VideoID:    "dQw4w9WgXcQ",
		Title:      "Song Title",
		Artist:     "Some Artist",
		Album:      "Some Album",
		ArtworkURL: "https://lh3.googleusercontent.com/abc=w544-h544-l90-rj",
		StartMs:    1_700_000_000_000,
		EndMs:      1_700_000_200_000,
		Playing:    true,
	}
}

func TestPlayingSongBecomesListeningActivity(t *testing.T) {
	got, err := json.Marshal(ToActivity(song(), false))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":2,"details":"Song Title","status_display_type":2,` +
		`"details_url":"https://music.youtube.com/watch?v=dQw4w9WgXcQ","state":"Some Artist",` +
		`"timestamps":{"start":1700000000000,"end":1700000200000},` +
		`"assets":{"large_image":"https://lh3.googleusercontent.com/abc=w544-h544-l90-rj","large_text":"Some Album"},` +
		`"buttons":[{"label":"Listen on YouTube Music","url":"https://music.youtube.com/watch?v=dQw4w9WgXcQ"}]}`
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestPausedOrMissingSongClears(t *testing.T) {
	paused := song()
	paused.Playing = false
	if ToActivity(paused, false) != nil || ToActivity(nil, false) != nil {
		t.Fatal("expected nil activity")
	}
	if b, _ := json.Marshal(ToActivity(nil, false)); string(b) != "null" {
		t.Fatalf("nil activity encodes as %s", b)
	}
}

func TestBasicModeLeavesOutNewerFields(t *testing.T) {
	a := ToActivity(song(), true)
	if a.StatusDisplayType != 0 || a.DetailsURL != "" || a.Details != "Song Title" {
		t.Fatalf("unexpected basic activity: %+v", a)
	}
	b, _ := json.Marshal(a)
	if strings.Contains(string(b), "status_display_type") || strings.Contains(string(b), "details_url") {
		t.Fatalf("basic payload has newer fields: %s", b)
	}
}

func TestLiveShowsElapsedOnly(t *testing.T) {
	s := song()
	s.EndMs = 0
	if got := ToActivity(s, false).Timestamps; !reflect.DeepEqual(got, &Timestamps{Start: s.StartMs}) {
		t.Fatalf("got %+v", got)
	}
}

func TestArtworkFallsBackToThumbnail(t *testing.T) {
	const thumb = "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg"
	s := song()
	s.ArtworkURL = "https://x.com/" + strings.Repeat("a", 300)
	if got := ToActivity(s, false).Assets.LargeImage; got != thumb {
		t.Fatalf("too long: got %s", got)
	}
	s.ArtworkURL = "http://insecure/img.jpg"
	if got := ToActivity(s, false).Assets.LargeImage; got != thumb {
		t.Fatalf("not https: got %s", got)
	}
	s.ArtworkURL, s.VideoID = "", ""
	if got := ToActivity(s, false).Assets; got != nil {
		t.Fatalf("expected no assets, got %+v", got)
	}
}

func TestNoVideoIDMeansNoLink(t *testing.T) {
	s := song()
	s.VideoID = ""
	a := ToActivity(s, false)
	if a.DetailsURL != "" || a.Buttons != nil {
		t.Fatalf("got %+v", a)
	}
}

func TestFit(t *testing.T) {
	if got := Fit("A", 128); got != "A⠀" {
		t.Fatalf("short: %q", got)
	}
	if got := Fit("", 128); got != "⠀⠀" {
		t.Fatalf("empty: %q", got)
	}
	if got := Fit("  many \u00a0 spaces\t\n ", 128); got != "many spaces" {
		t.Fatalf("spaces: %q", got)
	}
	long := Fit(strings.Repeat("😀", 100), 128)
	if utf16Len(long) > 128 || !strings.HasSuffix(long, "…") || !utf8.ValidString(long) {
		t.Fatalf("emoji: %q (%d units)", long, utf16Len(long))
	}
	if got := utf16Len(Fit(strings.Repeat("ก", 200), 128)); got != 128 {
		t.Fatalf("thai: %d units", got)
	}
	if got := Fit("😀", 128); got != "😀" {
		t.Fatalf("one emoji is two UTF-16 units, no padding: %q", got)
	}
}

func decode(t *testing.T, s string) any {
	t.Helper()
	d := json.NewDecoder(strings.NewReader(s))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestParseTrackRejectsJunkAndCleans(t *testing.T) {
	for _, junk := range []string{`null`, `"song"`, `{"title":"   "}`, `[1]`, `{"title":5}`} {
		if got := ParseTrack(decode(t, junk)); got != nil {
			t.Fatalf("%s: expected nil, got %+v", junk, got)
		}
	}
	got := ParseTrack(decode(t, `{"title":" Hi ","videoId":"bad id","startMs":"x","playing":"yes"}`))
	want := &Track{Title: "Hi"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestParseTrackNumbers(t *testing.T) {
	got := ParseTrack(decode(t, `{"title":"x","startMs":1.5e12,"endMs":1e400,"playing":true,"videoId":"dQw4w9WgXcQ"}`))
	if got.StartMs != 1.5e12 || got.EndMs != 0 || !got.Playing || got.VideoID != "dQw4w9WgXcQ" {
		t.Fatalf("got %+v", got)
	}
	// Plain float64 numbers (no UseNumber) work too.
	if got := ParseTrack(map[string]any{"title": "x", "startMs": 42.0}); got.StartMs != 42 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseTrackCutsLongFields(t *testing.T) {
	got := ParseTrack(map[string]any{"title": strings.Repeat("😀", 600), "artworkUrl": strings.Repeat("a", 5000)})
	if utf16Len(got.Title) != 1024 || len(got.ArtworkURL) != 4096 {
		t.Fatalf("title %d units, artwork %d", utf16Len(got.Title), len(got.ArtworkURL))
	}
	odd := ParseTrack(map[string]any{"title": "a" + strings.Repeat("😀", 600)})
	if utf16Len(odd.Title) != 1023 || !utf8.ValidString(odd.Title) {
		t.Fatalf("does not split a surrogate pair: %d units", utf16Len(odd.Title))
	}
}

func TestTimestampsRound(t *testing.T) {
	s := song()
	s.StartMs, s.EndMs = 1000.5, 2000.4
	if got := ToActivity(s, false).Timestamps; got.Start != 1001 || got.End != 2000 {
		t.Fatalf("got %+v", got)
	}
}
