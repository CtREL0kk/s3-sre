
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username      text NOT NULL,
    email         text NOT NULL,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_username_uidx ON users (username);
CREATE UNIQUE INDEX users_email_uidx ON users (email);

CREATE TYPE object_type AS ENUM ('folder', 'file');
CREATE TYPE object_visibility AS ENUM ('private', 'public');
CREATE TYPE grant_permission AS ENUM ('read', 'write');

CREATE TABLE objects (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    parent_id    uuid REFERENCES objects (id) ON DELETE CASCADE,
    name         text NOT NULL,
    type         object_type NOT NULL,
    visibility   object_visibility NOT NULL DEFAULT 'private',
    storage_key  text,
    size_bytes   bigint,
    content_type text,
    etag         text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT objects_name_unique UNIQUE (owner_id, parent_id, name),
    CONSTRAINT objects_not_own_parent CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX objects_owner_parent_idx ON objects (owner_id, parent_id);

CREATE TABLE object_grants (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id   uuid NOT NULL REFERENCES objects (id) ON DELETE CASCADE,
    grantee_id  uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permission  grant_permission NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT object_grants_unique UNIQUE (object_id, grantee_id)
);

CREATE INDEX object_grants_grantee_idx ON object_grants (grantee_id);

CREATE TABLE refresh_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX refresh_tokens_hash_uidx ON refresh_tokens (token_hash);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);