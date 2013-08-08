SOURCES=hlsprobe2.go source-loader.go stream-monitor.go http-client.go http-api.go stats.go reports.go logger.go helpers.go templates.go

hlsprobe2: $(SOURCES)
	go build $(SOURCES)
build: hlsprobe2
paxbuild: hlsprobe2
# use sudo or run as root
	paxctl -cm hlsprobe2
run: $(SOURCES)
	go run $(SOURCES)
install: hlsprobe2
# use sudo or run as root
	strip hlsprobe2
	cp -a hlsprobe2 /usr/local/bin
clean:
	rm hlsprobe2
