-- corti: healthcare appointment management.
-- Every row is scoped to an organization (tenant). See DESIGN.md for the
-- multi-tenancy / sharding strategy.
CREATE SCHEMA IF NOT EXISTS doodle;

-- TODO: Create tenacy for users if they belong to org. Can be used to derive from JWT.
-- CREATE TABLE doodle.organizations (
--     organization_id uuid PRIMARY KEY,
--     slug text NOT NULL UNIQUE,
--     region text NOT NULL DEFAULT 'eu',
--     created_at timestamptz NOT NULL DEFAULT NOW()
-- );

CREATE TABLE doodle.users (
    user_id uuid PRIMARY KEY,
    -- organization_id uuid NOT NULL REFERENCES doodle.organizations (organization_id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_org ON doodle.doctors (organization_id);

-- Timeslot availability.
CREATE TABLE doodle.timeslots (
    timeslot_id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES doodle.users (user_id) ON DELETE CASCADE,
    start_at timestamptz NOT NULL,
    end_at timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'available',
    created_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_timeslot_period CHECK (end_at > start_at),
    CONSTRAINT valid_timeslot_status CHECK (status IN ('available', 'booked'))
);
CREATE INDEX idx_timeslots_available ON doodle.timeslots (user_id, status, start_at);

CREATE TABLE doodle.meetings (
    meeting_id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES doodle.users (user_id) ON DELETE CASCADE,
    timeslot_id uuid NOT NULL UNIQUE REFERENCES doodle.timeslots (timeslot_id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'booked',
    start_at timestamptz NOT NULL,
    end_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_appointment_status CHECK (status IN ('booked', 'cancelled', 'completed'))
);
CREATE INDEX idx_meetings_user ON doodle.meetings (user_id, start_at);

-- Users can book only one meeting per timeslot_id - at most one active (ie booked) appointment.
CREATE UNIQUE INDEX uq_active_meeting_per_timeslot
ON doodle.meetings (user_id, timeslot_id)
WHERE status = 'booked';
