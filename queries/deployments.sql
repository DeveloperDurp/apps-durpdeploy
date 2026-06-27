-- name: ListDeploymentsByRelease :many
SELECT * FROM deployments WHERE release_id = ? ORDER BY created_at DESC;

-- name: GetDeployment :one
SELECT * FROM deployments WHERE id = ?;

-- name: CreateDeployment :one
INSERT INTO deployments (release_id, environment_id, status, started_at, finished_at) VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateDeployment :one
UPDATE deployments SET release_id = ?, environment_id = ?, status = ?, started_at = ?, finished_at = ? WHERE id = ? RETURNING *;

-- name: UpdateDeploymentStatus :exec
UPDATE deployments SET status = ?, started_at = ?, finished_at = ? WHERE id = ?;

-- name: ListDeployments :many
SELECT * FROM deployments ORDER BY created_at DESC;

-- name: ListRecentDeployments :many
SELECT * FROM deployments ORDER BY created_at DESC LIMIT ?;

-- name: DeleteDeployment :exec
DELETE FROM deployments WHERE id = ?;
