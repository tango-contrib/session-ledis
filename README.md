session-ledis [![Build Status](https://drone.io/github.com/tango-contrib/session-ledis/status.png)](https://drone.io/github.com/tango-contrib/session-ledis/latest) [![](http://gocover.io/_badge/github.com/tango-contrib/session-ledis)](http://gocover.io/github.com/tango-contrib/session-ledis)
======

Session-ledis is a store of [session](https://github.com/tango-contrib/session) middleware for [Tango](https://github.com/lunny/tango) stored session data via [ledis](http://ledisdb.com/). 

## Installation

    go get github.com/tango-contrib/session-ledis

## Simple Example

```Go
package main

import (
    "github.com/lunny/tango"
    "github.com/tango-contrib/session"
    "github.com/tango-contrib/session-ledis"
)

type SessionAction struct {
    session.Session
}

func (a *SessionAction) Get() string {
    a.Session.Set("test", "1")
    return a.Session.Get("test").(string)
}

func main() {
    o := tango.Classic()
    o.Use(session.New(session.Options{
        Store: redistore.New(ledistore.Options{
                Host:    "127.0.0.1",
                DbIndex: 0,
                MaxAge:  30 * time.Minute,
            }),
        }))
    o.Get("/", new(SessionAction))
}
```

## Getting Help

- [API Reference](https://gowalker.org/github.com/tango-contrib/session-ledis)
