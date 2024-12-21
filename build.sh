#!/usr/bin/env bash

for package in common bluez-mqtt pulseaudio-mqtt rotel-mqtt routeros-mqtt samsungtv-mqtt snapcast-mqtt cec-mqtt; do
  go build ./$package/...
done
