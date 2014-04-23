SOURCES=streamsurfer.go structure.go config.go monitor.go monitor-prober.go http-client.go http-api.go stats.go  logger.go zabbix.go helpers.go templates.go webui-report.go analyzer.go # reports.go source-loader.go
HTML=html/*.html
LDFLAGS="-X main.build_date `date -u +%Y%m%d%H%M%S`"

streamsurfer: $(SOURCES) $(HTML)
	go build -ldflags $(LDFLAGS) $(SOURCES)
gcc: $(SOURCES) $(HTML)
	go build -o streamsurfer -compiler gccgo -ldflags $(LDFLAGS) $(SOURCES)
gccbuild: gcc
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
