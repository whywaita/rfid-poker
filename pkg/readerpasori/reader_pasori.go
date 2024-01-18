package readerpasori

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/whywaita/rfid-poker/pkg/reader"

	"github.com/bamchoh/pasori"
	"github.com/google/gousb"
	"golang.org/x/sync/errgroup"
)

var (
	VID uint16 = 0x054C // SONY
	PID uint16 = 0x06C1 // RC-S380
)

func GetCard() ([]byte, error) {
	idm, err := pasori.GetID(VID, PID)
	if err != nil {
		return nil, fmt.Errorf("pasori.GetID(): %w", err)
	}
	return idm, nil
}

// PollingDevices polls all connected devices.
func PollingDevices(ch chan reader.Data) error {
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Define the USB device's Vendor and Product IDs
	vid, pid := gousb.ID(VID), gousb.ID(PID)
	// Find all devices with the specified Vendor and Product IDs
	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == vid && desc.Product == pid
	})
	if err != nil {
		return fmt.Errorf("ctx.OpenDevices(): %w", err)
	}
	defer func() {
		for _, dev := range devices {
			dev.Close()
		}
	}()

	log.Println("found devices:", len(devices))

	var eg errgroup.Group
	for _, dev := range devices {
		dev := dev
		eg.Go(func() error {
			serial, err := dev.SerialNumber()
			if err != nil {
				return fmt.Errorf("dev.SerialNumber(): %w", err)
			}

			for {
				uid, err := pasori.GetIDByDevice(ctx, dev)
				if err != nil {
					return fmt.Errorf("pasori.GetIDByDevice(): %w", err)
				}

				fmt.Printf("Device %s: Successfully processed (uid: %v)\n", serial, hex.EncodeToString(uid))
				ch <- reader.Data{
					UID:          uid,
					SerialNumber: serial,
				}

				time.Sleep(time.Millisecond * 500)
			}
		})

	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("eg.Wait(): %w", err)
	}

	return nil
}
