package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/whywaita/rfid-poker/pkg/query"
)

func GetAntennaBySerial(ctx context.Context, conn *sql.DB, deviceID string, antennaID int) (*query.GetAntennaBySerialRow, error) {
	q := query.New(conn)
	antenna, err := q.GetAntennaBySerial(ctx, ToSerial(deviceID, antennaID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("GetAntennaBySerial(): %w", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("antenna not found: %w", err)
	}

	return &antenna, nil
}

// RegisterNewDevice register new device to database
// serial is device serial number
// We become unknown as new antenna, that will be registered as new player, muck, board, etc.
func RegisterNewDevice(ctx context.Context, conn *sql.DB, deviceID string, pairID int) error {
	s := ToSerial(deviceID, pairID)
	unknownId, err := GetUnknownAntennaTypeID(ctx, conn)
	if err != nil {
		return fmt.Errorf("GetUnknownAntennaID(): %w", err)
	}

	q := query.New(conn)
	if err := q.AddNewAntenna(ctx, query.AddNewAntennaParams{
		Serial:        s,
		AntennaTypeID: unknownId,
	}); err != nil {
		return fmt.Errorf("AddNewAntenna(): %w", err)
	}

	return nil
}

func ToSerial(deviceID string, pairID int) string {
	return fmt.Sprintf("%s-%d", deviceID, pairID)
}

func FromSerial(serial string) (string, int, error) {
	s := strings.Split(serial, "-")
	if len(s) != 2 {
		return "", 0, errors.New("invalid serial format")
	}

	pairID, err := strconv.Atoi(s[1])
	if err != nil {
		return "", 0, fmt.Errorf("strconv.Atoi(): %w", err)
	}

	return s[0], pairID, nil
}

// GetUnknownAntennaTypeID get unknown antenna type id
func GetUnknownAntennaTypeID(ctx context.Context, conn *sql.DB) (int64, error) {
	q := query.New(conn)
	antennaTypeID, err := q.GetAntennaTypeIdIsUnknown(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetAntennaByAntennaTypeName(): %w", err)
	}

	return antennaTypeID, nil
}
