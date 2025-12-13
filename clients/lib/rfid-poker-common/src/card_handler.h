#ifndef CARD_HANDLER_H
#define CARD_HANDLER_H

#include <Arduino.h>

/**
 * Abstract interface for handling card detection and error events.
 * Implementations define how to send card data (HTTP, Serial, etc.)
 */
class CardHandler {
public:
  virtual ~CardHandler() {}

  /**
   * Called when a card is detected on a channel
   * @param channel Channel number (0-based)
   * @param uid Card UID as hex string (e.g., "04 AA BB CC DD 11")
   */
  virtual void onCardDetected(int channel, const char* uid) = 0;

  /**
   * Called when an error occurs
   * @param code Error code (e.g., "rfid_init_failed")
   * @param message Human-readable error message
   */
  virtual void onError(const char* code, const char* message) = 0;

  /**
   * Called on device boot
   * @param reason Boot reason (e.g., "power_on", "watchdog_reset")
   */
  virtual void onBoot(const char* reason) = 0;
};

#endif // CARD_HANDLER_H
