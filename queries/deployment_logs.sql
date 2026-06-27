-- name: ListDeploymentLogsByDeployment :many
SELECT * FROM deployment_logs WHERE deployment_id = ? ORDER BY created_at DESC;

-- name: GetDeploymentLog :one
SELECT * FROM deployment_logs WHERE id = ?;

-- name: CreateDeploymentLog :one
INSERT INTO deployment_logs (deployment_id, step_name, line) VALUES (?, ?, ?) RETURNING *;

-- name: UpdateDeploymentLog :one
UPDATE deployment_logs SET deployment_id = ?, step_name = ?, line = ? WHERE id = ? RETURNING *;

-- name: DeleteDeploymentLog :exec
DELETE FROM deployment_logs WHERE id = ?;
