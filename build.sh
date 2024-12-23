#!/usr/bin/env bash

for package in common bluez-mqtt pulseaudio-mqtt rotel-mqtt routeros-mqtt samsungtv-mqtt snapcast-mqtt cec-mqtt hid-mqtt mpd-mqtt; do
  go build ./$package/...
done
