ALTER TABLE participants
    ADD COLUMN IF NOT EXISTS voice_profile_id UUID;

ALTER TABLE voice_profiles
    ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS display_name TEXT;

ALTER TABLE voice_profiles
    ALTER COLUMN participant_id DROP NOT NULL;

UPDATE voice_profiles vp
SET owner_user_id = w.owner_user_id,
    display_name = COALESCE(NULLIF(vp.display_name, ''), p.name)
FROM participants p
JOIN workspaces w ON w.workspace_id = p.workspace_id
WHERE vp.participant_id = p.participant_id
  AND (vp.owner_user_id IS NULL OR vp.display_name IS NULL OR vp.display_name = '');

UPDATE voice_profiles
SET display_name = COALESCE(NULLIF(display_name, ''), 'Voice profile')
WHERE display_name IS NULL OR display_name = '';

UPDATE participants p
SET voice_profile_id = vp.voice_profile_id
FROM voice_profiles vp
WHERE vp.participant_id = p.participant_id
  AND p.voice_profile_id IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'participants_voice_profile_id_fkey'
    ) THEN
        ALTER TABLE participants
            ADD CONSTRAINT participants_voice_profile_id_fkey
            FOREIGN KEY (voice_profile_id)
            REFERENCES voice_profiles(voice_profile_id)
            ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS voice_profiles_owner_user_id_idx ON voice_profiles(owner_user_id);
CREATE INDEX IF NOT EXISTS participants_voice_profile_id_idx ON participants(voice_profile_id);
