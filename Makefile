build-cli: satcli-linux-amd64 satcli-linux-arm64 satcli-windows-amd64.exe satcli-windows-arm64.exe satcli-darwin-arm64 satcli-darwin-amd64

dg-sat:
	go build -o bin/$@ github.com/foundriesio/dg-satellite/cmd/server

satcli-linux-amd64:
satcli-linux-arm64:
satcli-windows-amd64.exe:
satcli-windows-arm64.exe:
satcli-darwin-amd64:
satcli-darwin-arm64:
satcli-%:
	CGO_ENABLED=0 \
	GOOS=$(shell echo $* | cut -f1 -d\- ) \
	GOARCH=$(shell echo $* | cut -f2 -d\- | cut -f1 -d. ) \
		go build -tags nodb -o bin/$@ github.com/foundriesio/dg-satellite/cmd/cli

