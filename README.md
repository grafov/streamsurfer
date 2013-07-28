HLS Probe Utility
=================

**The code doesn't work now! Development in progress**

Utility to detect errors in HTTP Live Streams (Apple HLS).
It may be used as regular monitoring tool and for mediaserver stress testing.
Features are:

 * parse M3U8-playlists (variant and single-bitrate playlists supported)
 * detect bad playlists format (empty playlists, incorrect chunk durations)
 * check HTTP response statuses and webserver timeouts

Planned features:

 * probe chunks with `mediainfo` utility (from libav)

This utility can't be used for HLS playback.

`hlsprobe2` is a port of Python `hlsprobe` (https://github.com/grafov/hlsprobe).

Install
-------

### Use binary

TODO Instructions follows later.

### Build from sources

You need Go language (http://golang.org) environment installed.
Then:

				go get https://github.com/grafov/m3u8
				git clone https://github.com/grafov/hlsprobe2
				cd hlsprobe2
				sudo make install

Similar projects
----------------

 * https://code.google.com/p/hls-player
 * https://github.com/brookemckim/hlspider

Project status
--------------

[![Build Status](https://travis-ci.org/grafov/hlsprobe2.png?branch=master)](https://travis-ci.org/grafov/hlsprobe2)

[![Is maintained?](http://stillmaintained.com/grafov/hlsprobe2.png)](http://stillmaintained.com/grafov/hlsprobe2)
