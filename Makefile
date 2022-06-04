build:
	mkdir -p build
	CGO_ENABLED=0 \
		go build -o ./build/lightsshd ./


release:
	strip ./build/lightsshd
	upx ./build/lightsshd

build-arm:
	GOOS=linux GOARCH=arm CGO_ENABLED=0 \
		go build -o ./build/lightsshd-linux-arm ./

release-arm:
	strip ./build/lightsshd-linux-arm || true
	upx ./build/lightsshd-linux-arm

.PHONY: build