Stream Surfer
=============

**The code is experimental. Undocumented. Not completed. Though it works for me.**
**I not recommend to use it until stable version published.**

Stream Surfer — probe utiliy for HTTP video streaming. The utility detects errors in
HTTP Streams (Apple HLS now supported) and monitor health of any HTTP resources. It may
be used as regular monitoring tool and stress testing for mediaservers (and any
HTTP-servers too).

Features are:

 * parse M3U8-playlists (master and single-bitrate playlists)
 * detect bad playlists format (empty playlists, incorrect chunk durations)
 * check HTTP response statuses
 * collects response time statistics
 * webreports to represent collected statistics
 * integration with Zabbix (http://zabbix.com) monitoring software

Planned features:

 * probe more formats beside HLS (planned HDS, DASH, Widevine VOD)
 * probe chunks with `mediainfo` utility (from ffmpeg)
 * REST HTTP to represent collected data and utility control
 * aggregate and analyze statistics from other streamsurfer nodes
 * visualize monitoring information in Web UI
 * persistent storage for statistics and reports generation

This software can't be used for HLS playback.

`streamsurfer` is an furfer development of Python `hlsprobe` (https://github.com/grafov/hlsprobe).

Install
-------

You need Go language (http://golang.org) environment installed.
Then:

	go get github.com/grafov/bcast
	go get github.com/grafov/m3u8
	go get github.com/gorilla/mux
	git clone https://github.com/grafov/streamsurfer
	cd streamsurfer
	sudo make install

The code includes Bootstrap 2 (http://getbootstrap.com) library (under Apache License).
It may be packaged with `streamsurfer` due GPLv3 license.
To simplify installation Bootstrap code yet included in `streamsurfer` package.
Later it will be splitted and Bootstrap will be downloaded separately.

Usage
-----

Setup configuration file (copy one of templates from package) and start utility:

    streamsurfer --config=config.yml

All stream problems logged to error log (`error-log` parameter in the config `params` section).
Web reports available at `localhost:8088` (define listener with `http-api-listen`).

Similar projects
----------------

 * http://code.google.com/p/hls-player
 * http://github.com/brookemckim/hlspider

Build status
------------

[![Build Status](https://travis-ci.org/grafov/hlsprobe2.png?branch=master)](https://travis-ci.org/grafov/hlsprobe2)

[![Is maintained?](http://stillmaintained.com/grafov/hlsprobe2.png)](http://stillmaintained.com/grafov/hlsprobe2)
