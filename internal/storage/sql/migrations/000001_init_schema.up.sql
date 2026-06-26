CREATE SCHEMA IF NOT EXISTS doodle;

CREATE TABLE doodle.users (
    user_id uuid PRIMARY KEY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

-- Timeslot availability
CREATE TABLE doodle.timeslots (
    timeslot_id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES doodle.users (user_id) ON DELETE CASCADE,
    start_at timestamptz NOT NULL,
    end_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_timeslot_period CHECK (end_at > start_at)
);
CREATE INDEX idx_timeslots_available ON doodle.timeslots (user_id, start_at);

-- Meetings for user creation and participation from others
-- TODOs to consider:
-- Create max participants
-- Create notifictations
CREATE TABLE doodle.meetings (
    meeting_id uuid PRIMARY KEY,
    creator_user_id uuid NOT NULL REFERENCES doodle.users (user_id) ON DELETE CASCADE, -- creator
    timeslot_id uuid NOT NULL UNIQUE REFERENCES doodle.timeslots (timeslot_id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'booked',
    start_at timestamptz NOT NULL,
    end_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_meeting_status CHECK (status IN ('booked', 'cancelled', 'completed'))
);
CREATE INDEX idx_meetings_creator_user ON doodle.meetings (creator_user_id, start_at);

CREATE TABLE doodle.meeting_participants (
    user_id uuid NOT NULL REFERENCES doodle.users (user_id),
    meeting_id uuid NOT NULL REFERENCES doodle.meetings (meeting_id)
);
CREATE INDEX idx_meetings_participants ON doodle.meeting_participants (user_id, meeting_id);

-- User creators can only create one meeting per timeslot_id where the status is booked
-- Another constraint could be that they cant create meetings in the past
CREATE UNIQUE INDEX uq_booked_meeting_per_timeslot_per_creator_user
ON doodle.meetings (creator_user_id, timeslot_id)
WHERE status = 'booked';

-- Users can book only create one meeting per timeslot_id where the status is booked
CREATE UNIQUE INDEX uq_active_meeting_per_timeslot_per_user
ON doodle.meeting_participants (user_id, meeting_id);
