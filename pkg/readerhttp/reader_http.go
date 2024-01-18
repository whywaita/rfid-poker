package readerhttp

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/whywaita/rfid-poker/pkg/reader"
)

func PollingHTTP(ch chan reader.Data) error {
	http.HandleFunc("/card", func(w http.ResponseWriter, r *http.Request) {
		HandleCards(w, r, ch)
	})

	log.Printf("Start HTTP server... (port: 8081)")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		return fmt.Errorf("http.ListenAndServe(): %w", err)
	}

	return nil
}

type Card struct {
	UID    string `json:"uid"`
	Serial string `json:"serial"`
}

func HandleCards(w http.ResponseWriter, r *http.Request, ch chan reader.Data) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Printf("invalid method: %s", r.Method)
		return
	}

	input := Card{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("invalid request body: %v", err)
		return
	}

	uid, err := hex.DecodeString(input.UID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("invalid UID: %v", err)
		return
	}

	ch <- reader.Data{
		UID:          uid,
		SerialNumber: input.Serial,
	}

	w.WriteHeader(http.StatusOK)
}
