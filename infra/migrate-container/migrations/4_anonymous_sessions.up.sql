CREATE TABLE IF NOT EXISTS anonymous_sessions (
    session_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audio_id      UUID NOT NULL,
    owner_user_id UUID NOT NULL REFERENCES users(user_id),
    status        TEXT NOT NULL CHECK(status IN ('processing','ready','confirmed','expired')) DEFAULT 'processing',
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMP NOT NULL DEFAULT NOW() + INTERVAL '7 days'
);

CREATE TABLE IF NOT EXISTS anonymous_speakers (
    speaker_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID NOT NULL REFERENCES anonymous_sessions(session_id) ON DELETE CASCADE,
    label        TEXT NOT NULL,
    fragment_key TEXT NOT NULL,
    segments     JSONB NOT NULL DEFAULT '[]',
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS anon_sessions_owner_idx  ON anonymous_sessions(owner_user_id);
CREATE INDEX IF NOT EXISTS anon_sessions_audio_idx  ON anonymous_sessions(audio_id);
CREATE INDEX IF NOT EXISTS anon_speakers_session_idx ON anonymous_speakers(session_id);
