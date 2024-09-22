-- name: CreateData :exec
INSERT INTO data (id, type, fileName, filePath, burn, expire, passwordHash, passwordSalt, encryptSalt)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetDataByID :one
SELECT type, fileName, filePath, burn, passwordHash, passwordSalt, encryptSalt FROM data WHERE id = ?;

-- name: DeleteDataByID :exec
DELETE FROM data WHERE id = ?;

-- name: ListExpired :many
SELECT id, filePath FROM data WHERE expire <= ?;

-- name: HasDataWithID :one
SELECT CAST(EXISTS(SELECT id FROM data WHERE id = ?) AS bool);