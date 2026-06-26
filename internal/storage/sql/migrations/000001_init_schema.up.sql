CREATE SCHEMA IF NOT EXISTS doodle;

CREATE TABLE doodle.users (
    user_id uuid PRIMARY KEY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

-- Timeslot availability owned by a single user (their calendar).
CREATE TABLE doodle.timeslots (
    timeslot_id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES doodle.users (user_id) ON DELETE CASCADE,
    start_at timestamptz NOT NULL,
    end_at timestamptz NOT NULL,
    duration_mins int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT min_duration CHECK (duration_mins >= 10),
    CONSTRAINT max_duration CHECK (duration_mins <= 60),
    CONSTRAINT valid_timeslot_period CHECK (end_at > start_at)
);

-- Primary access path: a user's slots ordered by time, used by the
-- "allotted" and "calendar" range queries.
CREATE INDEX idx_timeslots_user_start ON doodle.timeslots (user_id, start_at);

-- A booked meeting occupies exactly one timeslot. The UNIQUE constraint on timeslot_id is the concurrency guard, since two racing meetings for the same slot cannot both succeed. Loser gets a unique-violation (mapped to 409).
CREATE TABLE doodle.meetings (
    meeting_id uuid PRIMARY KEY,
    timeslot_id uuid NOT NULL UNIQUE REFERENCES doodle.timeslots (timeslot_id) ON DELETE CASCADE,
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    url text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT NOW()
);

-- Participants are plain emails (no auth, anyone can be invited).
CREATE TABLE doodle.meeting_participants (
    meeting_id uuid NOT NULL REFERENCES doodle.meetings (meeting_id) ON DELETE CASCADE,
    attendee_email text NOT NULL,
    PRIMARY KEY (meeting_id, attendee_email)
);
