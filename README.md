Lesser
======

`lesser` is a minimal reimplementation of the standard `less` pager.  It aims to
be very fast, particularly for searching the file.

`lesser` is very much a work-in-progress, but supports the most basic features.

The currently supported keybindings are as follows:

Control:

* `q`: Quit

Scrolling:

* `j`: Scroll down
* `k`: Scroll up
* `g`: Scroll to top
* `G`: Scroll to bottom
* `Pgdn`: Scroll down one screen full
* `Pgup`: Scroll up one screen full
* `^D`: Scroll down one half screen full
* `^U`: Scroll up one half screen full

Searching:

* `/`: Enter search regex (re2 syntax). Press enter to search.
* `n`: Jump down to next search result
* `N`: Jump up to previous search result
