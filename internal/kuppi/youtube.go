package kuppi

import (
	"net/url"
	"strings"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
)

func ParseYouTubeURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" {
		return "", apperror.WithCode(apperror.ErrInvalid, "invalid_youtube_url")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host != "youtube.com" && host != "www.youtube.com" && host != "m.youtube.com" && host != "youtu.be" {
		return "", apperror.WithCode(apperror.ErrInvalid, "unsupported_video_provider")
	}
	var id string
	switch host {
	case "youtu.be":
		if parsed.Path != "/" {
			id = strings.TrimPrefix(parsed.Path, "/")
		}
	default:
		switch {
		case parsed.Path == "/watch":
			id = parsed.Query().Get("v")
		case strings.HasPrefix(parsed.Path, "/embed/"):
			id = strings.TrimPrefix(parsed.Path, "/embed/")
		case strings.HasPrefix(parsed.Path, "/shorts/"):
			id = strings.TrimPrefix(parsed.Path, "/shorts/")
		}
	}
	if len(id) != 11 || strings.Contains(id, "/") || !validVideoID(id) {
		return "", apperror.WithCode(apperror.ErrInvalid, "invalid_youtube_video_id")
	}
	return id, nil
}

func validVideoID(id string) bool {
	for _, r := range id {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return len(id) == 11
}
