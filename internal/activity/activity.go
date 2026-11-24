// Package activity turns the song from the extension into a Discord activity
// payload. Everything coming from the browser is treated as untrusted and
// validated here.
//
// Lengths follow JavaScript (UTF-16 code units), because that is how the
// extension and Discord count characters.
package activity

import (
	"encoding/json"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Track has the same shape as `Track` in extension/types.d.ts.
type Track struct {
	VideoID    string  `json:"videoId"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Album      string  `json:"album"`
	ArtworkURL string  `json:"artworkUrl"`
	StartMs    float64 `json:"startMs"`
	EndMs      float64 `json:"endMs"`
	Playing    bool    `json:"playing"`
}

// Activity is the subset of Discord's activity object this app sends.
type Activity struct {
	Type              int         `json:"type"`
	Details           string      `json:"details"`
	StatusDisplayType int         `json:"status_display_type,omitempty"`
	DetailsURL        string      `json:"details_url,omitempty"`
	State             string      `json:"state,omitempty"`
	Timestamps        *Timestamps `json:"timestamps,omitempty"`
	Assets            *Assets     `json:"assets,omitempty"`
	Buttons           []Button    `json:"buttons,omitempty"`
}

// Timestamps drive Discord's time bar. Floats so huge values encode like JavaScript.
type Timestamps struct {
	Start float64 `json:"start"`
	End   float64 `json:"end,omitempty"`
}

type Assets struct {
	LargeImage string `json:"large_image"`
	LargeText  string `json:"large_text,omitempty"`
}

type Button struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

const (
	activityListening    = 2
	statusDisplayDetails = 2   // member list shows "Listening to <song>" instead of the app name
	maxText              = 128 // Discord's limit for details/state/large_text
	maxImageURL          = 256 // Discord's limit for an asset key or URL
	dayMs                = 24 * 60 * 60 * 1000
)

var videoIDPattern = regexp.MustCompile(`^[\w-]{11}$`)

// ParseTrack validates a track decoded from JSON (with UseNumber or plain
// float64 numbers). Anything that isn't a song with a title gives nil.
func ParseTrack(input any) *Track {
	o, ok := input.(map[string]any)
	if !ok {
		return nil
	}
	title := trimJS(str(o["title"], 1024))
	if title == "" {
		return nil
	}
	videoID := str(o["videoId"], 1024)
	if !videoIDPattern.MatchString(videoID) {
		videoID = ""
	}
	playing, _ := o["playing"].(bool)
	return &Track{
		VideoID:    videoID,
		Title:      title,
		Artist:     trimJS(str(o["artist"], 1024)),
		Album:      trimJS(str(o["album"], 1024)),
		ArtworkURL: str(o["artworkUrl"], 4096),
		StartMs:    num(o["startMs"]),
		EndMs:      num(o["endMs"]),
		Playing:    playing,
	}
}

// str is the string value cut to max UTF-16 units, or "" if it isn't a string.
func str(value any, max int) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return sliceUTF16(s, max)
}

// num is the finite number value, or 0.
func num(value any) float64 {
	var f float64
	switch v := value.(type) {
	case float64:
		f = v
	case json.Number:
		parsed, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return 0 // out of range: JavaScript would see Infinity
		}
		f = parsed
	default:
		return 0
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return f
}

// Fit collapses whitespace and fits Discord's 2..max character rule without
// splitting emoji.
func Fit(text string, max int) string {
	clean := trimJS(collapseSpace(text))
	out := clean
	if utf16Len(clean) > max {
		var b strings.Builder
		n := 0
		for _, r := range clean {
			w := runeUnits(r)
			if n+w > max-1 {
				break
			}
			b.WriteRune(r)
			n += w
		}
		b.WriteString("…")
		out = b.String()
	}
	for utf16Len(out) < 2 {
		out += "\u2800" // braille blank "⠀": Discord rejects 1-character strings
	}
	return out
}

func fit(text string) string { return Fit(text, maxText) }

func isHTTPSURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && u.Host != ""
}

func imageFor(t *Track) string {
	if utf16Len(t.ArtworkURL) <= maxImageURL && isHTTPSURL(t.ArtworkURL) {
		return t.ArtworkURL
	}
	if t.VideoID != "" {
		return "https://i.ytimg.com/vi/" + t.VideoID + "/hqdefault.jpg"
	}
	return ""
}

func timestampsFor(t *Track) *Timestamps {
	if t.StartMs <= 0 {
		return nil
	}
	start := jsRound(t.StartMs)
	length := t.EndMs - t.StartMs
	if length > 0 && length < dayMs {
		return &Timestamps{Start: start, End: jsRound(t.EndMs)}
	}
	return &Timestamps{Start: start}
}

// ToActivity builds the payload for a playing track; nil clears the status.
// basic leaves out the newer fields (clickable song title, song name in the
// member list), for Discord versions that reject them.
func ToActivity(t *Track, basic bool) *Activity {
	if t == nil || !t.Playing {
		return nil
	}
	songURL := ""
	if t.VideoID != "" {
		songURL = "https://music.youtube.com/watch?v=" + t.VideoID
	}

	a := &Activity{Type: activityListening, Details: fit(t.Title)}
	if !basic {
		a.StatusDisplayType = statusDisplayDetails
		a.DetailsURL = songURL
	}
	if t.Artist != "" {
		a.State = fit(t.Artist)
	}
	a.Timestamps = timestampsFor(t)
	if image := imageFor(t); image != "" {
		a.Assets = &Assets{LargeImage: image}
		if t.Album != "" {
			a.Assets.LargeText = fit(t.Album)
		}
	}
	if songURL != "" {
		a.Buttons = []Button{{Label: "Listen on YouTube Music", URL: songURL}}
	}
	return a
}

// jsRound is Math.round: halves round towards +Infinity.
func jsRound(x float64) float64 { return math.Floor(x + 0.5) }

// isJSSpace matches JavaScript's \s (and what String.prototype.trim removes).
func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', '\u00a0', '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000', '\ufeff':
		return true
	}
	return r >= '\u2000' && r <= '\u200a'
}

func trimJS(s string) string { return strings.TrimFunc(s, isJSSpace) }

// collapseSpace is text.replace(/\s+/g, ' ').
func collapseSpace(s string) string {
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if isJSSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
			}
			inSpace = true
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}
	return b.String()
}

func runeUnits(r rune) int {
	if r >= 0x10000 && r <= utf8.MaxRune {
		return 2
	}
	return 1
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += runeUnits(r)
	}
	return n
}

// sliceUTF16 keeps the first max UTF-16 units. Unlike JavaScript it never
// leaves half of a surrogate pair behind.
func sliceUTF16(s string, max int) string {
	n := 0
	for i, r := range s {
		w := runeUnits(r)
		if n+w > max {
			return s[:i]
		}
		n += w
	}
	return s
}
