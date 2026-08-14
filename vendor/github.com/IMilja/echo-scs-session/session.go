// Package session provides SCS (https://github.com/alexedwards/scs) session
// middleware for Echo v5 (https://github.com/labstack/echo).
//
// All credit for the underlying session engine goes to Alex Edwards
// (https://github.com/alexedwards), the author of the scs package.
package session

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// Config defines the config for the session middleware.
type Config struct {
	// Skipper defines a function to skip middleware.
	Skipper middleware.Skipper

	// SessionManager is the scs session manager used to load and save
	// session data. Required.
	SessionManager *scs.SessionManager
}

// DefaultConfig is the default session middleware config.
var DefaultConfig = Config{
	Skipper: middleware.DefaultSkipper,
}

// LoadAndSave provides middleware which automatically loads and saves
// session data for the current request, and communicates the session token
// to and from the client in a cookie.
func LoadAndSave(sessionManager *scs.SessionManager) echo.MiddlewareFunc {
	config := DefaultConfig
	config.SessionManager = sessionManager

	return LoadAndSaveWithConfig(config)
}

// LoadAndSaveWithConfig returns a session middleware using the provided
// config.
func LoadAndSaveWithConfig(config Config) echo.MiddlewareFunc {
	if config.Skipper == nil {
		config.Skipper = DefaultConfig.Skipper
	}

	if config.SessionManager == nil {
		panic("session: middleware requires a session manager")
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			ctx := c.Request().Context()

			var token string
			if cookie, err := c.Cookie(config.SessionManager.Cookie.Name); err == nil {
				token = cookie.Value
			}

			ctx, err := config.SessionManager.Load(ctx, token)
			if err != nil {
				return err
			}

			c.SetRequest(c.Request().WithContext(ctx))

			// Echo v5's Context.Response() returns the plain
			// http.ResponseWriter, so the echo.Response wrapper (which
			// exposes the Before hook we need to write the session
			// cookie after the handler has run but before the response
			// is flushed) has to be recovered via UnwrapResponse.
			res, err := echo.UnwrapResponse(c.Response())
			if err != nil {
				return err
			}

			res.Before(func() {
				if config.SessionManager.Status(ctx) == scs.Unmodified {
					// session might not exist yet, nothing to do
					return
				}

				responseCookie := &http.Cookie{
					Name:     config.SessionManager.Cookie.Name,
					Path:     config.SessionManager.Cookie.Path,
					Domain:   config.SessionManager.Cookie.Domain,
					Secure:   config.SessionManager.Cookie.Secure,
					HttpOnly: config.SessionManager.Cookie.HttpOnly,
					SameSite: config.SessionManager.Cookie.SameSite,
				}

				switch config.SessionManager.Status(ctx) {
				case scs.Modified:
					token, expiry, err := config.SessionManager.Commit(ctx)
					if err != nil {
						panic(err)
					}

					if config.SessionManager.GetBool(ctx, "__rememberMe") {
						responseCookie.Expires = time.Unix(expiry.Unix()+1, 0)        // Round up to the nearest second.
						responseCookie.MaxAge = int(time.Until(expiry).Seconds() + 1) // Round up to the nearest second.
					}

					responseCookie.Value = token

				case scs.Destroyed:
					responseCookie.Expires = time.Unix(1, 0)
					responseCookie.MaxAge = -1
				}

				c.SetCookie(responseCookie)
				addHeaderIfMissing(c.Response(), "Cache-Control", `no-cache="Set-Cookie"`)
				addHeaderIfMissing(c.Response(), "Vary", "Cookie")
			})

			return next(c)
		}
	}
}

func addHeaderIfMissing(w http.ResponseWriter, key, value string) {
	for _, h := range w.Header()[key] {
		if h == value {
			return
		}
	}
	w.Header().Add(key, value)
}
