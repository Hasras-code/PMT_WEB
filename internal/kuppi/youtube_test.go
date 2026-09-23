package kuppi

import "testing"

func TestParseYouTubeURL(t *testing.T) {
	valid := []string{
		"https://www.youtube.com/watch?v=abcdefghijk",
		"https://youtube.com/watch?v=abcdefghijk&list=playlist",
		"https://m.youtube.com/embed/abcdefghijk",
		"https://youtu.be/abcdefghijk",
		"https://www.youtube.com/shorts/abcdefghijk?feature=share",
	}
	for _, raw := range valid {
		got, err := ParseYouTubeURL(raw)
		if err != nil || got != "abcdefghijk" {
			t.Errorf("ParseYouTubeURL(%q) = %q, %v", raw, got, err)
		}
	}
	invalid := []string{
		"http://youtu.be/abcdefghijk",
		"https://youtube.com/playlist?list=abcdefghijk",
		"https://evil-youtube.com/watch?v=abcdefghijk",
		"https://youtube.com.evil.example/watch?v=abcdefghijk",
		"https://youtube.com/watch?v=short",
		"<iframe src=\"https://youtube.com/embed/abcdefghijk\">",
		"javascript:https://youtube.com/watch?v=abcdefghijk",
	}
	for _, raw := range invalid {
		if _, err := ParseYouTubeURL(raw); err == nil {
			t.Errorf("ParseYouTubeURL(%q) unexpectedly succeeded", raw)
		}
	}
}
