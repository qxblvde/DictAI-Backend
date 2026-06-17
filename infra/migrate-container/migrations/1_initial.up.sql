CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS workspaces (
    workspace_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(user_id),
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS participants (
    participant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    UNIQUE(workspace_id, email)
);

CREATE TABLE IF NOT EXISTS voice_profiles (
    voice_profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id UUID NOT NULL REFERENCES participants(participant_id) ON DELETE CASCADE,
    embedding VECTOR(192),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS results (
    result_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audio_id       UUID NOT NULL,
    workspace_id   UUID NOT NULL,
    upload_user_id UUID NOT NULL,
    summary_key    TEXT NOT NULL,
    transcript_key TEXT NOT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS meetings (
    meeting_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
    audio_id UUID NOT NULL,
    transcript JSONB,
    speakers JSONB,
    structured_transcript JSONB,
    summary TEXT,
    status TEXT NOT NULL CHECK(status IN ('pending','processed','error')) DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
