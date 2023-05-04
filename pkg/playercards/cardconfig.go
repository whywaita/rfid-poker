package playercards

import (
	"encoding/hex"
	"fmt"
)

func LoadPlayerCard(in []byte, cardIDs map[string]string) (string, error) {
	str := hex.EncodeToString(append(in[:], 0x00, 0x00, 0x00))

	v, ok := cardIDs[str]
	if !ok {
		return "", fmt.Errorf("unknown card")
	}

	return v, nil
}
