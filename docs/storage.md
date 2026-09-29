# Object storage

The API supports local storage for development and Cloudflare R2 for production. Select the provider explicitly with `STORAGE_PROVIDER=local` or `STORAGE_PROVIDER=r2`. Production configuration rejects the local provider because Cloud Run filesystems are ephemeral.

## Buckets and configuration

R2 uses its S3-compatible API through AWS SDK for Go v2. The backend requires `R2_ENDPOINT`, `R2_REGION` (normally `auto`), `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_PRIVATE_BUCKET`, `R2_PUBLIC_BUCKET`, and `R2_PUBLIC_BASE_URL`. The public base URL must be a Cloudflare public or custom domain; it is not the authenticated R2 S3 endpoint.

Keep `R2_ACCESS_KEY_ID` and `R2_SECRET_ACCESS_KEY` in Google Secret Manager. Never add them to frontend or `VITE_*` configuration.

Private storage contains resource PDFs, announcement attachments, fund receipts, profile images, and event covers. Event covers remain private even for public events; the public event route verifies that the event is published and public before returning a short-lived signed URL. The public bucket contains batch hero images and published gallery variants.

## Upload flow

1. The authenticated frontend sends filename, MIME type, and size to the existing feature upload endpoint.
2. The API checks feature and cohort authorization, generates an immutable object key, records a pending upload intent, and returns a ten-minute presigned PUT URL.
3. The browser uploads directly to R2 with the exact returned `Content-Type` header.
4. The browser calls the returned authenticated `confirm_url`.
5. The API uses `HeadObject` and a bounded ranged read to verify size, MIME type, and PDF/image content.
6. The upload intent becomes `UPLOADED` and can be consumed by the existing feature mutation.

The database stores only stable object keys. Presigned URLs and R2 credentials are never persisted or logged.

New R2 objects use server-generated keys such as:

```text
batches/{batchID}/resources/uploads/{uuid}.pdf
batches/{batchID}/resources/{resourceID}/versions/{uuid}.pdf
batches/{batchID}/announcements/{announcementID}/attachments/{uuid}.pdf
batches/{batchID}/funds/{fundID}/transactions/{transactionID}/receipts/{uuid}.pdf
users/{userID}/profile/{uuid}.{ext}
batches/{batchID}/profile/{uuid}.{ext}
batches/{batchID}/events/{eventID}/cover/{uuid}.{ext}
gallery/uploads/{uuid}.{ext}
```

Original filenames remain separate database metadata and are used only for display and download disposition.

## Downloads and public media

Private download endpoints continue to authenticate the caller and perform cohort or domain authorization. They then return a five-minute presigned R2 GET URL, so Cloud Run does not proxy the bytes.

Gallery responses return public media URLs derived from `R2_PUBLIC_BASE_URL`. Existing public batch image routes redirect to the public object. Local development retains the current API-served behavior.

## R2 CORS

Direct browser PUT requests require bucket CORS. Configure the private and public buckets with the exact development and production frontend origins:

```json
[
  {
    "AllowedOrigins": [
      "http://localhost:5173",
      "https://YOUR-PAGES-OR-CUSTOM-DOMAIN"
    ],
    "AllowedMethods": ["PUT"],
    "AllowedHeaders": ["Content-Type"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3600
  }
]
```

Replace the placeholder with the real Cloudflare Pages or application domain. Ordinary image embedding and top-level download navigation do not need additional CORS methods.

## Existing local objects

Migration `000025_object_storage` records each upload intent's provider and public/private class without changing historical object keys. Do not remove `data/files` before migration.

Inventory first:

```sh
make migrate-storage-dry-run
```

Then copy and verify referenced objects:

```sh
make migrate-storage
```

The command is restartable. It uploads a local object, verifies it in R2, and only then changes that upload intent to provider `R2`. Already verified objects are reused. Local copies are retained for rollback and must be removed only through a separate reviewed cleanup after production verification.

Expired, unconsumed objects are removed by `cmd/jobs` from the provider recorded on their upload intent. Consumed historical files are retained according to their domain archive policy.
