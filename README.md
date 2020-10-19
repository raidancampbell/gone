# gone
A simple leadership election service for distributed systems.  
This solves the problem of distributed schedulers executing at the same time, when only one execution is desired.

This is an HTTP wrapper over redis's global locks, with HTTP GET requests for ease of use.  
`gone` itself is safe to be scaled up and still return a single leader, thanks to the guarantees of `redsync`. 

## Installation
Spin up a redis instance, listening at `localhost:6379`
```shell script
docker run -p 6379:6379 --name rds -d redis
```
Run the code, the HTTP server runs on port 8080 by default.
```shell script
go run main.go
```

## Usage
No user set up required, keys are assumed to never collide.
1. Choose a system identifier.  This self-chosen string is reused across your distributed service: e.g. if there are 5 redundant copies of a "cleaning scheduler" service, then this would be something like `cleaning_scheduler`.
1. Choose a round identifier.  This self-chosen string is ephemeral, but must match across your distributed service: e.g. if a tmpfs cleaner runs hourly, this would be something like `2020-10-18-0800-tmpfscleaner`.  All copies of the redundant service must generate the same value, so coarse timestamps are recommended.
1. Whoever receives the `202 Accepted` HTTP response wins!  Losers receive `204 No Content`.
1. Optional: upon completion, the winner sends a POST to the `complete` endpoint to indicate successful execution.

### Am I the leader?
HTTP GET to `/api/v1/elect/{system_identifier}/{round_identifier}`
 - Returns a `202 Accepted` if the requester is the elected leader
 - Returns a `204 No Content` if the requester is not the elected leader. These requests will take longer (approx. 1.5s) due to mutex timeouts.
 - In case of `500 Internal Server Error`, an error occurred when recording or verifying leadership.  It is safe to retry.

### Who was the leader?
HTTP GET to `/api/v1/elected/{system_identifier}/{round_identifier}`
 - Returns a `200 OK` alongside the `Request.RemoteAddr` of the elected leader
 - If no leader was elected (e.g. the round hasn't occurred yet), `204 No Content` is returned.
 - In case of `500 Internal Server Error`, a `redis` communication error has occurred.  It is safe to retry. 

### optional: Did I do my job?
HTTP GET `/api/v1/complete/{system_identifier}/{round_identifier}`
 - Returns a `200 OK` upon successful recording
 - Returns a `400 Bad Request` if another leader(!!!) has already recorded their completion.  If this is encountered, the calling system is in a bad state.
 - In case of `500 Internal Server Error`, an error occurred when recording or verifying the completion.  It is safe to retry.

### optional: Did anyone do the job?
HTTP GET `/api/v1/completed/{system_identifier}/{round_identifier}`
 - Returns a `204 No Content` if no completion found for the given round
 - Returns a `200 OK` alongside the `Request.RemoteAddr` of the completer
 - In case of `500 Internal Server Error`, a `redis` communication error has occurred.  It is safe to retry.