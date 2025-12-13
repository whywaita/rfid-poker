#ifndef HTTP_CARD_HANDLER_H
#define HTTP_CARD_HANDLER_H

#include <Arduino.h>
#include <card_handler.h>

/**
 * CardHandler implementation that sends data via HTTP POST
 */
class HttpCardHandler : public CardHandler {
public:
  HttpCardHandler(const String& deviceId, const String& apiHost);

  void onCardDetected(int channel, const char* uid) override;
  void onError(const char* code, const char* message) override;
  void onBoot(const char* reason) override;

  String getDeviceId() const { return _deviceId; }
  String getApiHost() const { return _apiHost; }

private:
  String _deviceId;
  String _apiHost;

  void postCard(const String& uid, int pairId);
  int getPairID(int channelId);
};

#endif // HTTP_CARD_HANDLER_H
