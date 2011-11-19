README for gockel
=================
Andreas Krennmair <ak@synflood.at>

Introduction
------------

Gockel is a Twitter client for text terminals, written in the Go programming language.
Currently, gockel is at a very early stage of development. Its predecessor is
baconbird: http://synflood.at/baconbird.html

Downloading
-----------

You can download gockel here: https://github.com/akrennmair/gockel

Building
--------

In order to build gockel, you can simply run "gomake" or "gb". But be aware that
gockel requires a number of external packages that you need to install beforehand.

In particular, gockel depends upon the following packages:

* github.com/akrennmair/goauth
* github.com/akrennmair/go-stfl (you need to install libstfl before building it)
* goconf.googlecode.com/hg

Use goinstall to install these packages.

If you want to install gockel into your path, run "gomake install" or "gb -i". To
uninstall, run "gomake nuke" or "gb -N".

Using
-----

Simply run the gockel binary. When starting for the first time, you will be asked
to open a URL, where you need to confirm that gockel is authorized to read your
tweets and post updates. You will then be shown a PIN code that you need to enter
into gockel. After that, gockel starts up as usual, and downloads the latest
updates in your home timeline. On subsequent starts, you will not be asked
for a PIN code again.

Currently, the following keystrokes are available:

* q: quit program
* ENTER: write a new tweet
* Ctrl-R: retweet currently selected tweet
* r: reply to currently selected tweet
* Ctrl-F: favorite currently selected tweet
* Ctrl-O: shorten all URLs in the current input field
* F: follow a user (you will be prompted for the username)
* U: unfollow the publisher user of the currently selected tweet
* 1..9: select user (for multi-account support, see below)

Configuration
-------------

If you want to tweak gockel, you need to create a configuration file under
the path ~/.gockel/gockelrc in an INI-style format. The examples directory 
contains several examples upon which you can build. General options are
in the default section (i.e. those options with no section marker), the
user interface color are in the "colors" section, and special highlighting
configuration can be found in sections starting with "highlight" (see below).

Currently, the following configuration options are available:

* http_timeout: default connection timeout for HTTP requests in seconds (default: 60 seconds)
* default_user: set the user that is selected by default on startup (for multi-account support, see below)
* ignore_incoming: set the user(s) (space-separated list) for which no data from their timeline
  shall be fetched. This feature was added to configure "write-only" accounts that are not
  interested in reading the timeline.

Color Configuration
-------------------

The color configuration is put into a configuration section named "colors".
The color configuration string is a comma-separated list of key=value pairs.  
Every configuration string can contain at most one "fg" key (to define the 
foreground color), at most one "bg" key (to define the background color), and 
optionally one or more "attr" keys (to define extended attributes).

The following color names are available:

* black
* red
* green
* yellow
* blue
* magenta
* cyan
* white
* color0 .. color255 (requires 256-color terminal emulator, e.g. xterm with TERM=xterm-256color)

The following attributes are available:

* standout
* underline
* reverse
* blink
* dim
* bold
* protect
* invis

Please note that not all attributes may be supported by your terminal emulator.

The following user interface elements can be configured color-wise:

* shorthelp: short help line
* infotext: informational text line
* listfocus: focused line in the tweet list
* listnormal: unfocused lines in the tweet list
* background: general application background
* input: input field
* userlist: user list (top line)
* userlist_active: active selection in user list

### Example ###

	[colors]
	shorthelp = fg=white,bg=red
	infotext = bg=white,fg=red
	listfocus = fg=white,bg=green,attr=bold
	listnormal = fg=yellow

Highlighting
------------

In addition it is also possible to configure pairs of regular expressions and
color configuration strings to highlight certain text in the tweet list. For
each highlighting, you need to configure an own configuration section whose
name starts with "highlight" and which must be unique. Such a section needs
to contain two configuration options, "attributes", which contains a color
configuration string (see above), and "regex", which contains the regular
expression that describes what shall be highlighted. If you regular expression
starts with a special character such as '#' (comment in INI-style configuration
files), you can mark start and end of regular expression with forward slashes.

### Example ###

	[highlight_urls]
	attributes = fg=green
	regex = (mailto|ftp|https?):[^ )\]]*

Multiple Accounts
-----------------

If you need to manage multiple accounts, gockel provides basic support for 
that. When you start gockel for the first time, you need to authenticate your 
user. To add more accounts, run "gockel -add" and you will be provided with
the same workflow to authorize the application. Gockel will save this 
information in ~/.gockel as files starting with "access_token.json", and 
usually the associated Twitter username as suffix.

In the application, you can then select the currently active user by pressing 
the keys 1 to 9. Which user is currently active is displayed in the user list 
line at the top of the application. When you start gockel, the first user
that was found is active, but you can influence this by configuring

	default_user = your_preferred_nick

in your ~/.gockel/gockelrc.


Contact
-------
Andreas Krennmair <ak@synflood.at>

License
-------
Gockel is licensed under the MIT/X Consortium License. See the file LICENSE
for further details.
