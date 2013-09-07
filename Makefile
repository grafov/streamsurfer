SOURCES=streamsurfer.go structures.go source-loader.go stream-monitor.go http-client.go http-api.go stats.go reports.go logger.go zabbix.go helpers.go templates.go
LDFLAGS=-ldflags "-X main.build_date `date -u +%Y%m%d%H%M%S`"
streamsurfer: $(SOURCES)
	go build $(LDFLAGS) $(SOURCES)
build: streamsurfer
paxbuild: streamsurfer
# use sudo or run as root
	paxctl -cm streamsurfer
run: $(SOURCES)
	go run $(SOURCES)
install: streamsurfer
# use sudo or run as root
	strip streamsurfer
	cp -a streamsurfer /usr/local/bin
clean:
	rm streamsurfer
