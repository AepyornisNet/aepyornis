# echo-scs-session

[![Go Reference](https://pkg.go.dev/badge/github.com/IMilja/echo-scs-session.svg)](https://pkg.go.dev/github.com/IMilja/echo-scs-session)

SCS session middleware for [Echo v5](https://github.com/labstack/echo).

All credit for the underlying session engine goes to [Alex Edwards](https://github.com/alexedwards),
author of the [scs](https://github.com/alexedwards/scs) package this middleware wraps.

This package is a port of [garnizeh/echo-scs-session](https://github.com/garnizeh/echo-scs-session)
(itself a fork of [canidam/echo-scs-session](https://github.com/canidam/echo-scs-session) and
[spazzymoto/echo-scs-session](https://github.com/spazzymoto/echo-scs-session)) updated for Echo v5.

## Why a separate package

Echo v5 changed `echo.Context` from an interface to a concrete struct and, more importantly,
`Context.Response()` now returns a plain `http.ResponseWriter` instead of Echo's own `*echo.Response`
wrapper. The `Before` hook that this middleware relies on to write the session cookie after the
handler runs (but before the response is flushed) only exists on `*echo.Response`, so it now has to
be recovered with `echo.UnwrapResponse(c.Response())`. That, plus the `echo.HandlerFunc`/
`echo.MiddlewareFunc` signatures moving to `func(c *echo.Context) error`, is enough of a breaking
change that the v4 middleware can't be used as-is against v5.

## Installation

This package requires Go 1.24+ and Echo v5.

```
go get github.com/IMilja/echo-scs-session
```

## Basic use

See [scs](https://github.com/alexedwards/scs) for the full range of configuration options
(session stores, timeouts, 'remember me', etc).

```go
package main

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/labstack/echo/v5"

	session "github.com/IMilja/echo-scs-session"
)

var sessionManager *scs.SessionManager

func main() {
	// Initialize a new session manager and configure the session lifetime.
	sessionManager = scs.New()
	sessionManager.Lifetime = 24 * time.Hour

	e := echo.New()

	// Use the LoadAndSave() middleware.
	e.Use(session.LoadAndSave(sessionManager))

	e.GET("/put", putHandler)
	e.GET("/get", getHandler)

	e.Logger.Fatal(e.Start(":4000"))
}

func putHandler(c *echo.Context) error {
	// Store a new key and value in the session data.
	sessionManager.Put(c.Request().Context(), "message", "Hello from a session!")

	return c.String(http.StatusOK, "")
}

func getHandler(c *echo.Context) error {
	// Use the GetString helper to retrieve the string value associated with a
	// key. The zero value is returned if the key does not exist.
	msg := sessionManager.GetString(c.Request().Context(), "message")

	return c.String(http.StatusOK, msg)
}
```

```
$ curl -i --cookie-jar cj --cookie cj localhost:4000/put
HTTP/1.1 200 OK
Cache-Control: no-cache="Set-Cookie"
Content-Type: text/plain; charset=UTF-8
Set-Cookie: session=0KumL8V5WYuvZwEQj2IPrYvm-cC3y7m8xQWLhTmxq_U; Path=/; HttpOnly; SameSite=Lax
Vary: Cookie
Date: Fri, 07 Aug 2026 08:28:00 GMT
Content-Length: 0

$ curl -i --cookie-jar cj --cookie cj localhost:4000/get
HTTP/1.1 200 OK
Content-Type: text/plain; charset=UTF-8
Date: Fri, 07 Aug 2026 08:28:05 GMT
Content-Length: 22

Hello from a session!
```

## Custom config

`LoadAndSaveWithConfig` accepts a `Skipper` if you need to bypass the middleware for
certain requests:

```go
e.Use(session.LoadAndSaveWithConfig(session.Config{
	SessionManager: sessionManager,
	Skipper: func(c *echo.Context) bool {
		return c.Path() == "/healthz"
	},
}))
```

## License

MIT — see [LICENSE](LICENSE).
