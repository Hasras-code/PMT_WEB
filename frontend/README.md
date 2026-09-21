# PMT LMS frontend

React, TypeScript, Vite and Tailwind portal for the PMT LMS API.

## Local development

Start PostgreSQL, migrations and the API from the repository root, then run:

```sh
npm install
npm run dev
```

The development server runs at `http://localhost:5173`. API requests default to `http://localhost:8080`; set `VITE_API_URL` when the API is hosted elsewhere. Keep the frontend origin in `CORS_ALLOWED_ORIGINS` on the API.

Useful checks:

```sh
npm run build
npm run lint
```

The client uses short-lived bearer access tokens and serialized refresh-token rotation. Management controls are driven by `/v1/me/access`, while the API remains responsible for authorization on every request.
