module github.com/claes/routeros-mqtt

go 1.21

require (
	github.com/eclipse/paho.mqtt.golang v1.4.3
	github.com/go-routeros/routeros/v3 v3.0.0
	github.com/jfreymuth/pulse v0.1.1
)

require (
	github.com/claes/mqtt-bridges/common v0.0.0-00010101000000-000000000000 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
)

replace github.com/claes/mqtt-bridges/common => ../common
