package user

import (
	"context"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
)

func (s Service) Image(ctx context.Context, user string) (upload.Object, error) {
	var o upload.Object
	e := s.Pool.QueryRow(ctx, `SELECT i.storage_key,i.file_name,i.mime_type FROM users u JOIN upload_intents i ON i.storage_key=u.profile_image_key WHERE u.id=$1`, user).Scan(&o.Key, &o.Name, &o.MIME)
	return o, db.Error(e)
}
