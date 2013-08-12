HLS Probe Utility
=================

Utility to detect errors in HTTP Live Streams (Apple HLS).
It may be used as regular monitoring tool and for mediaserver stress testing.
Features are:

 * parse M3U8-playlists (master and single-bitrate playlists)
 * detect bad playlists format (empty playlists, incorrect chunk durations)
 * check HTTP response statuses and webserver timeouts
 * response time statistics
 * webreports to represent collected statistics
 * integration with Zabbix monitoring software

**The code in alpha. Mostly works but not all complete.**

Planned features:

 * probe chunks with `mediainfo` utility (from ffmpeg)
 * REST HTTP to represent collected data and utility control
 * aggregate and analyze statistics from other hlsprobe nodes

This utility can't be used for HLS playback.

`hlsprobe2` is an improved port of Python `hlsprobe` (https://github.com/grafov/hlsprobe).

Install
-------

### Use binary

TODO Instructions follows later.

### Build from sources

You need Go language (http://golang.org) environment installed.
Then:

	go get github.com/grafov/m3u8
	go get github.com/gorilla/mux
	go get github.com/hoisie/mustache
	git clone https://github.com/grafov/hlsprobe2
	cd hlsprobe2
	sudo make install


Usage
-----

Setup configuration file (copy one of templates from package) and start utility:

    hlsprobe --config=config.yml

All stream problems logged to error log (`error-log` parameter in the config `params` section).
Web reports available at `localhost:8088` (define listener with `http-api-listen`).

Similar projects
----------------

 * https://code.google.com/p/hls-player
 * https://github.com/brookemckim/hlspider

Project status
--------------

[![Build Status](https://travis-ci.org/grafov/hlsprobe2.png?branch=master)](https://travis-ci.org/grafov/hlsprobe2)

[![Is maintained?](http://stillmaintained.com/grafov/hlsprobe2.png)](http://stillmaintained.com/grafov/hlsprobe2)
