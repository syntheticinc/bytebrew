-- name: CreateTeam :one
INSERT INTO teams (name, owner_id, max_seats)
VALUES ($1, $2, $3)
RETURNING id, name, owner_id, max_seats, created_at, updated_at;

-- name: GetTeamByID :one
SELECT id, name, owner_id, max_seats, created_at, updated_at
FROM teams
WHERE id = $1;

-- name: GetTeamByOwnerID :one
SELECT id, name, owner_id, max_seats, created_at, updated_at
FROM teams
WHERE owner_id = $1;

-- name: GetTeamByUserID :one
SELECT t.id, t.name, t.owner_id, t.max_seats, t.created_at, t.updated_at
FROM teams t
JOIN team_members tm ON tm.team_id = t.id
WHERE tm.user_id = $1;

-- name: UpdateTeamMaxSeats :exec
UPDATE teams SET max_seats = $1, updated_at = NOW() WHERE id = $2;

-- name: AddTeamMember :one
INSERT INTO team_members (team_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, team_id, user_id, role, joined_at;

-- name: RemoveTeamMember :exec
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: ListTeamMembers :many
SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.joined_at, u.email
FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = $1
ORDER BY tm.joined_at ASC;

-- name: CountTeamMembers :one
SELECT COUNT(*) FROM team_members WHERE team_id = $1;

-- name: CreateTeamInvite :one
INSERT INTO team_invites (team_id, email, invited_by, token, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, team_id, email, invited_by, token, status, created_at, expires_at;

-- name: GetTeamInviteByToken :one
SELECT id, team_id, email, invited_by, token, status, created_at, expires_at
FROM team_invites
WHERE token = $1;

-- name: UpdateInviteStatus :exec
UPDATE team_invites SET status = $1 WHERE id = $2;

-- name: ListPendingInvites :many
SELECT id, team_id, email, invited_by, token, status, created_at, expires_at
FROM team_invites
WHERE team_id = $1 AND status = 'pending'
ORDER BY created_at DESC;
