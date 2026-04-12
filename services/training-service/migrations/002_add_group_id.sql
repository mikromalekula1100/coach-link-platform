-- +goose Up
ALTER TABLE training_plans ADD COLUMN group_id UUID;
CREATE INDEX idx_plans_group ON training_plans (group_id) WHERE group_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_plans_group;
ALTER TABLE training_plans DROP COLUMN IF EXISTS group_id;
