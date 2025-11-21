#ifndef RFID_CORE_H
#define RFID_CORE_H

#include <Arduino.h>
#include <vector>
#include "card_handler.h"
#include "config.h"

/**
 * Core RFID reader logic shared between WiFi and Wired clients
 */
class RfidCore {
public:
  /**
   * Constructor
   * @param handler CardHandler implementation for sending card events
   */
  RfidCore(CardHandler* handler);

  /**
   * Initialize RFID readers
   * Must be called in setup()
   */
  void begin();

  /**
   * Read all RFID readers and send detected cards
   * Should be called in loop()
   */
  void update();

  /**
   * Check if a specific pair has all cards detected
   * @param pairId Pair ID (1-3)
   * @return true if all cards in the pair are detected
   */
  bool isPairComplete(int pairId) const;

  /**
   * Get list of pair IDs based on current configuration
   * @return Vector of pair IDs
   */
  std::vector<int> listPairID() const;

private:
  CardHandler* _handler;

  // Card detection status for each channel
  bool _cardsDetected[MAX_RFID_READERS];

  // Card history for debouncing
  struct CardHistory {
    String uid;
    unsigned long lastSentTime;
  };
  CardHistory _cardHistory[MAX_RFID_READERS];

  // Internal methods
  void selectChannel(uint8_t channel);
  String readUID();
  bool hasCard();
  int getPairID(int channelId) const;
  void triggerReadUID(int channel, const String& uid);
};

#endif // RFID_CORE_H
