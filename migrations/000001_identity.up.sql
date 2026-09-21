CREATE TABLE users (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), student_number text NOT NULL UNIQUE,
 first_name text NOT NULL, last_name text NOT NULL, display_name text NOT NULL,
 email text NOT NULL, password_hash text NOT NULL, phone_number text,
 profile_image_key text, email_verified_at timestamptz,
 status text NOT NULL DEFAULT 'PENDING_VERIFICATION' CHECK(status IN ('PENDING_VERIFICATION','ACTIVE','SUSPENDED','ARCHIVED')),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email ON users(lower(email));
CREATE TABLE auth_sessions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),user_id uuid NOT NULL REFERENCES users(id),
 refresh_token_hash bytea NOT NULL UNIQUE,transport text NOT NULL CHECK(transport IN ('token','cookie')),
 expires_at timestamptz NOT NULL, revoked_at timestamptz,created_at timestamptz NOT NULL DEFAULT now(),
 last_used_at timestamptz NOT NULL DEFAULT now(),user_agent text
);
CREATE INDEX sessions_user ON auth_sessions(user_id);
CREATE TABLE spent_refresh_tokens(token_hash bytea PRIMARY KEY,session_id uuid NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,used_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE verification_tokens(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),user_id uuid NOT NULL REFERENCES users(id),purpose text NOT NULL CHECK(purpose IN ('EMAIL_VERIFY','PASSWORD_RESET')),token_hash bytea NOT NULL UNIQUE,expires_at timestamptz NOT NULL,used_at timestamptz,created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX verification_user ON verification_tokens(user_id,purpose);
