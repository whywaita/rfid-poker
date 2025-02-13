-- name: GetAntenna :many
SELECT antenna.id, serial, antenna_type_id, player_id, antenna_type.name AS antenna_type_name
FROM antenna
JOIN antenna_type ON antenna_type.id = antenna.antenna_type_id;

-- name: GetAntennaBySerial :one
SELECT antenna.id, serial, antenna_type_id, player_id, antenna_type.name AS antenna_type_name
FROM antenna
JOIN antenna_type ON antenna_type.id = antenna.antenna_type_id
WHERE serial = ?;

-- name: AddNewAntenna :exec
INSERT INTO antenna (serial, antenna_type_id)
VALUES (?, ?);

-- name: SetPlayerIDToAntennaBySerial :exec
UPDATE antenna SET player_id = ?,
                   antenna_type_id = (SELECT id FROM antenna_type WHERE name = 'player')
WHERE serial = ?;

-- name: GetAntennaTypeIdIsUnknown :one
SELECT id FROM antenna_type WHERE name = 'unknown';

-- name: ResetAntenna :exec
DELETE FROM antenna;