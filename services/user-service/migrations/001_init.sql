-- +goose Up

-- Enable trigram extension for fuzzy search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- User profiles (synced from auth-service via NATS events)
CREATE TABLE user_profiles (
    id         UUID PRIMARY KEY,
    login      VARCHAR(50) NOT NULL UNIQUE,
    email      VARCHAR(255) NOT NULL,
    full_name  VARCHAR(255) NOT NULL,
    role       VARCHAR(20) NOT NULL CHECK (role IN ('coach', 'athlete')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_profiles_role ON user_profiles (role);
CREATE INDEX idx_user_profiles_full_name_trgm ON user_profiles USING gin (full_name gin_trgm_ops);
CREATE INDEX idx_user_profiles_login_trgm ON user_profiles USING gin (login gin_trgm_ops);

-- Connection requests between athletes and coaches
CREATE TABLE connection_requests (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    athlete_id UUID NOT NULL REFERENCES user_profiles(id),
    coach_id   UUID NOT NULL REFERENCES user_profiles(id),
    status     VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (athlete_id, coach_id)
);

CREATE INDEX idx_connection_requests_coach_id ON connection_requests (coach_id);
CREATE INDEX idx_connection_requests_athlete_id ON connection_requests (athlete_id);
CREATE INDEX idx_connection_requests_status ON connection_requests (status);

-- Active coach-athlete relations (one coach per athlete)
CREATE TABLE coach_athlete_relations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id   UUID NOT NULL REFERENCES user_profiles(id),
    athlete_id UUID NOT NULL UNIQUE REFERENCES user_profiles(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_coach_athlete_relations_coach_id ON coach_athlete_relations (coach_id);

-- Training groups owned by coaches
CREATE TABLE training_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id   UUID NOT NULL REFERENCES user_profiles(id),
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_training_groups_coach_id ON training_groups (coach_id);

-- Group membership (many-to-many between groups and athletes)
CREATE TABLE training_group_members (
    group_id   UUID NOT NULL REFERENCES training_groups(id) ON DELETE CASCADE,
    athlete_id UUID NOT NULL REFERENCES user_profiles(id),
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, athlete_id)
);

CREATE INDEX idx_training_group_members_athlete_id ON training_group_members (athlete_id);

-- +goose Down

DROP TABLE IF EXISTS training_group_members;
DROP TABLE IF EXISTS training_groups;
DROP TABLE IF EXISTS coach_athlete_relations;
DROP TABLE IF EXISTS connection_requests;
DROP TABLE IF EXISTS user_profiles;
DROP EXTENSION IF EXISTS pg_trgm;
