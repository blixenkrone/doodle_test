# Doodle tech test

## How to run
- `$ docker compose up -d --build`
- Visit localhost:8080/docs/index.html for HTTP API
- Visit localhost:5050 for a postgres admin interface (opt)
Host: postgres
DB: doodle
User: doodle
Pass: doodle

For an easy smoke test/data seed, run:
```sh
chmod +x ./doodle_smoke.sh
./doodle_smoke.sh
```

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

1 POST /users  
Create a new user for onboarding purposes

POST /timeslots  
Create a new timeslot 
- Could consider an expires_at property
- Max duration of 60 mins

GET /timeslots/allotted  
Get allotted/created timeslots as a user

GET /timeslots/calendar  
See calendar

PATCH /timeslots/{id}  
Update timeslot
- Constraint: Cant be done if there's a meeting booked at the time

DELETE /timeslots/{id}  
- Constraint: Cant be done if there's a meeting booked at the time

POST /timeslots/meeting  
- Create meeting

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


## Closing args
- I didn't get to properly implement the marking time slots as busy or free as a status type property. I would've implemented a simple endpoint for this ensuring the same constraints as updating the timeslot (as already implemented).
