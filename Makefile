SOURCES=streamsurfer.go structures.go source-loader.go stream-monitor.go http-client.go http-api.go stats.go reports.go logger.go zabbix.go helpers.go templates.go

streamsurfer: $(SOURCES)
	go build $(SOURCES)
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
