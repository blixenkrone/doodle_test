# Doodle tech test
 
## Architecture
- RDB storage solution
- REST API
- Docker based
  - Binary
  - Storage dep
  - Healthcheck
- Git repo
- GH actions to build, test, "deploy"
- Integration test
- Unit test
- Metrics


## API design

1. Create a user
POST /users
{name}

1. Create available timeslot
POST /timeslots
{
  availability: [{availability_time_start, availability_time_to}],
  duration_mins,
}
Docs:
- Could consider an expires_at property
- Max duration of 60 mins

2. Get allotted timeslots:
GET /timeslots/allotted?date_start=XYZ&date_end=DOE
[{  id, time_start, duration, is_booked  }]
Docs:
- Constraint: Filter out past timeslots

3. Update/delete timeslot
Docs:
- Constraint: Cant be done if there's a meeting booked at the time

4. Create meeting:
POST /timeslots/meeting
{  id, title, descr, attendees, url },
Docs:
- Constraint: No auth, anyone can book anything.
- Future: Make it shareable with only certain emails

5. See personal calendar
GET /timeslots/calendar?
[{ id, datetime_start, datetime_end, attendees }]


## Implementation details
- Avoid clashing meetings, ie. data races

## Notes/Constraints
- Insecure user management
- No authentication considered
- Sharable URLs are void
- No TZ, all is UTC
- The description was a little vague in this area, but I assumed that the users doing time slot management, were not the same users, that would do a meeting scheduling.

## TODO
- API endpoint docs
- Make mermaid diagram
- Implement solution

## Future considerations
- Meeting reminders
- Max meetings per day within a timeslot to allow for breaks etc.

