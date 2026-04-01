-- +goose Up

CREATE TABLE training_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    scheduled_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_plans_coach ON training_plans(coach_id);
CREATE INDEX idx_plans_date ON training_plans(scheduled_date);

CREATE TABLE training_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES training_plans(id) ON DELETE CASCADE,
    athlete_id UUID NOT NULL,
    coach_id UUID NOT NULL,
    athlete_full_name VARCHAR(255) NOT NULL,
    athlete_login VARCHAR(50) NOT NULL,
    coach_full_name VARCHAR(255) NOT NULL DEFAULT '',
    coach_login VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'assigned' CHECK (status IN ('assigned', 'completed', 'archived')),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ
);
CREATE INDEX idx_assignments_coach ON training_assignments(coach_id, status);
CREATE INDEX idx_assignments_athlete ON training_assignments(athlete_id, status);
CREATE INDEX idx_assignments_plan ON training_assignments(plan_id);

CREATE TABLE training_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL UNIQUE REFERENCES training_assignments(id) ON DELETE CASCADE,
    athlete_id UUID NOT NULL,
    content TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL,
    perceived_effort INTEGER NOT NULL CHECK (perceived_effort BETWEEN 0 AND 10),
    max_heart_rate INTEGER,
    avg_heart_rate INTEGER,
    distance_km DECIMAL(7,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reports_athlete ON training_reports(athlete_id);
CREATE INDEX idx_reports_assignment ON training_reports(assignment_id);

CREATE TABLE training_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_templates_coach ON training_templates(coach_id);

-- +goose Down

DROP TABLE IF EXISTS training_reports;
DROP TABLE IF EXISTS training_assignments;
DROP TABLE IF EXISTS training_plans;
DROP TABLE IF EXISTS training_templates;
