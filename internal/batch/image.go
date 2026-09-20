package batch

import (
	"context"
	"github.com/Hasras-code/PMT_WEB.git/internal/platform/db"
	"github.com/Hasras-code/PMT_WEB.git/internal/upload"
)

func (s Service) PublicImage(ctx context.Context, slug, eventID string) (upload.Object, error) {
	var o upload.Object
	var e error
	if eventID == "" {
		e = s.Pool.QueryRow(ctx, `SELECT i.storage_key,i.file_name,i.mime_type FROM batches b JOIN batch_profiles p ON p.batch_id=b.id JOIN upload_intents i ON i.storage_key=p.hero_image_key WHERE b.slug=$1 AND b.status<>'ARCHIVED'`, slug).Scan(&o.Key, &o.Name, &o.MIME)
	} else {
		e = s.Pool.QueryRow(ctx, `SELECT i.storage_key,i.file_name,i.mime_type FROM batches b JOIN events e ON e.batch_id=b.id JOIN upload_intents i ON i.storage_key=e.cover_image_key WHERE b.slug=$1 AND b.status<>'ARCHIVED' AND e.id=$2 AND e.status='PUBLISHED' AND e.visibility='PUBLIC'`, slug, eventID).Scan(&o.Key, &o.Name, &o.MIME)
	}
	return o, db.Error(e)
}
