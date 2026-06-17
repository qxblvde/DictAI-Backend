DROP INDEX IF EXISTS participants_voice_profile_id_idx;
DROP INDEX IF EXISTS voice_profiles_owner_user_id_idx;

ALTER TABLE participants
    DROP CONSTRAINT IF EXISTS participants_voice_profile_id_fkey;

ALTER TABLE participants
    DROP COLUMN IF EXISTS voice_profile_id;

ALTER TABLE voice_profiles
    DROP COLUMN IF EXISTS display_name,
    DROP COLUMN IF EXISTS owner_user_id;

ALTER TABLE voice_profiles
    ALTER COLUMN participant_id SET NOT NULL;
