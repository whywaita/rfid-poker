#ifndef SERIAL_CARD_HANDLER_H
#define SERIAL_CARD_HANDLER_H

#include <Arduino.h>
#include <ArduinoJson.h>
#include <card_handler.h>

/**
 * CardHandler implementation that sends JSON Lines format over Serial
 */
class SerialCardHandler : public CardHandler {
public:
  void onCardDetected(int channel, const char* uid) override;
  void onError(const char* code, const char* message) override;
  void onBoot(const char* reason) override;

  const char* getDeviceID();
  const char* getFirmwareVersion();

private:
  unsigned long _sequenceCounter = 0;

  String getTimestamp();
  unsigned long getNextSequence();
};

#endif // SERIAL_CARD_HANDLER_H
