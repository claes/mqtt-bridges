#!/usr/bin/env bash

# common
for package in common pulseaudio-mqtt rotel-mqtt routeros-mqtt samsungtv-mqtt snapcast-mqtt cec-mqtt; do
  go build ./$package/...
done
