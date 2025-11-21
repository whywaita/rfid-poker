#include <Arduino.h>
#include <FastLED.h>
#include <M5Atom.h>

#include <rfid_core.h>
#include <config.h>
#include "serial_card_handler.h"

// Global instances
SerialCardHandler cardHandler;
RfidCore rfidCore(&cardHandler);

void setup() {
  // Initialize Serial first for communication with Linux host
  Serial.begin(115200);
  while (!Serial) {
    delay(10); // Wait for serial port to connect
  }

  // Initialize M5Atom (enable Serial, disable I2C initially, enable display)
  M5.begin(true, false, true);

  // Display device information on serial (for debugging, not JSON Lines)
  Serial.println("# RFID Reader - M5Stack Wired Client");
  Serial.print("# Device ID: ");
  Serial.println(cardHandler.getDeviceID());
  Serial.print("# Client Type: ");
  Serial.println(getClientType());
  Serial.print("# RFID Reader Count: ");
  Serial.println(getRfidReaderCount());
  Serial.println("# Starting...");
  Serial.flush();

  // Send boot message via Serial in JSON Lines format
  cardHandler.onBoot("power_on");

  // Setup RFID readers using the common library
  rfidCore.begin();

  Serial.println("# RFID Reader initialized");
  Serial.flush();
}

void loop() {
  // Update RFID readers
  rfidCore.update();

  // Check if pair 1 is complete (we assume ATOM has 2 antennas = 1 pair)
  if (rfidCore.isPairComplete(1)) {
    // Light up the LED with green color when both antennas detect cards
    M5.dis.drawpix(0, 0x00ff00); // Green
  } else {
    // Turn off the LED when not all antennas detect cards
    M5.dis.drawpix(0, 0x000000); // Off
  }

  M5.update();
  delay(200); // Reduced from 1000ms for faster response
}
