-- name: CreateUser :one
INSERT INTO doodle.users (user_id, name)
VALUES ($1, $2)
RETURNING user_id, name, created_at;

-- name: CreateTimeslot :one
INSERT INTO doodle.timeslots (timeslot_id, user_id, start_at, end_at, duration_mins)
VALUES ($1, $2, $3, $4, $5)
RETURNING timeslot_id, user_id, start_at, end_at, duration_mins, created_at;

-- name: GetTimeslot :one
SELECT timeslot_id, user_id, start_at, end_at, duration_mins, created_at
FROM doodle.timeslots
WHERE timeslot_id = $1;

-- ListAllottedTimeslots returns a user's slots within a window, excluding any
-- that have already started, flagging which are booked.
-- name: ListAllottedTimeslots :many
SELECT
    t.timeslot_id,
    t.start_at,
    t.end_at,
    t.duration_mins,
    EXISTS (
        SELECT 1 FROM doodle.meetings m WHERE m.timeslot_id = t.timeslot_id
    ) AS is_booked
FROM doodle.timeslots t
WHERE t.user_id = $1
    AND t.start_at >= sqlc.arg(date_start)
    AND t.start_at < sqlc.arg(date_end)
    AND t.start_at >= sqlc.arg(now)
ORDER BY t.start_at;

-- DeleteTimeslot removes a slot only if it is not booked. The row count tells
-- the caller whether it was deleted (1) or blocked/absent (0).
-- name: DeleteTimeslot :execrows
DELETE FROM doodle.timeslots t
WHERE t.timeslot_id = $1
    AND NOT EXISTS (
        SELECT 1 FROM doodle.meetings m WHERE m.timeslot_id = t.timeslot_id
    );

-- UpdateTimeslot mutates a slot only if it is not booked.
-- name: UpdateTimeslot :execrows
UPDATE doodle.timeslots t
SET start_at = $2, end_at = $3, duration_mins = $4
WHERE t.timeslot_id = $1
    AND NOT EXISTS (
        SELECT 1 FROM doodle.meetings m WHERE m.timeslot_id = t.timeslot_id
    );

-- name: CreateMeeting :one
INSERT INTO doodle.meetings (meeting_id, timeslot_id, title, description, url)
VALUES ($1, $2, $3, $4, $5)
RETURNING meeting_id, timeslot_id, title, description, url, created_at;

-- name: AddMeetingParticipant :exec
INSERT INTO doodle.meeting_participants (meeting_id, attendee_email)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- ListCalendarMeetings returns the booked meetings on a user's calendar with
-- their attendee emails aggregated into an array.
-- name: ListCalendarMeetings :many
SELECT
    m.meeting_id,
    m.title,
    t.start_at,
    t.end_at,
    COALESCE(
        array_agg(p.attendee_email) FILTER (WHERE p.attendee_email IS NOT NULL),
        '{}'
    )::text[] AS attendees
FROM doodle.meetings m
INNER JOIN doodle.timeslots t ON t.timeslot_id = m.timeslot_id
LEFT JOIN doodle.meeting_participants p ON p.meeting_id = m.meeting_id
WHERE t.user_id = $1
GROUP BY m.meeting_id, m.title, t.start_at, t.end_at
ORDER BY t.start_at;
