# Kuppi recordings

Kuppis are module-scoped recorded teaching sessions backed by YouTube. They are separate from scheduled cohort events.

## API

The API is batch-scoped under `/v1/batches/{batchID}/kuppis`:

- `GET /kuppis` lists visible recordings.
- `POST /kuppis` creates a draft.
- `GET /kuppis/{kuppiID}` retrieves a recording.
- `PATCH /kuppis/{kuppiID}` updates a draft or published recording.
- `POST /kuppis/{kuppiID}/publish` publishes a draft.
- `POST /kuppis/{kuppiID}/archive` archives a draft or published recording.

Create and update requests use `youtube_url`, never iframe HTML or a raw video ID. Accepted URLs are HTTPS links on `youtube.com`, `www.youtube.com`, `m.youtube.com`, or `youtu.be` using watch, embed, shorts, or short-link forms.

Students see published Kuppis only. Batch representatives and academic representatives can manage Kuppis in their active cohort. Platform administrators are authorized through the existing platform administration capability.