/*

*/

const byte signature[] = {2, 4, 3, 4};
const int pinsUsed[] = {0, 1, 2, 3, 4, 5, 6, 7};
const int potentiometerPin = A10;
byte buttonState = 0;

void setup()
{
    for (int i = 0; i < 8; i++)
    {
        pinMode(pinsUsed[i], INPUT_PULLUP);
    }
}

void intByteArrayConv(int i, byte *buffer)
{
    buffer[0] = (i >> 8) & 0xFF;
    buffer[1] = i & 0xFF;
}

void loop()
{
    for (int i = 0; i < 8; i++)
    {
        int pinState = digitalRead(pinsUsed[i]);
        int prevState = bitRead(buttonState, i);

        if (pinState != prevState)
        {
            if (pinState == 1)
                bitSet(buttonState, i);
            else
                bitClear(buttonState, i);
        }
    }

    Serial.write(signature, 4);
    Serial.write(buttonState);
    // for(int i = 0; i < 8; i++) {
    //   Serial.write(bitRead(buttonState, i));
    // }

    Serial.write(map(analogRead(potentiometerPin), 0, 1023, 0, 100));
}
