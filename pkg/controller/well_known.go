package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

type WellKnownController interface {
	WebFinger(c echo.Context) error
	HostMeta(c echo.Context) error
}

type wellKnownController struct {
	context *container.Container
}

func NewWellKnownController(c *container.Container) WellKnownController {
	return &wellKnownController{context: c}
}

// WebFinger implementation based on https://github.com/go-ap/webfinger/blob/master/handlers.go
func (wc *wellKnownController) WebFinger(c echo.Context) error {
	res := c.QueryParam("resource")
	if res == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("missing resource parameter"))
	}

	typ, handle := splitResourceString(res)
	if typ == "" || handle == "" {
		return renderApiError(c, http.StatusBadRequest, fmt.Errorf("invalid resource: %s", res))
	}

	var host string
	switch typ {
	case "acct":
		if strings.Contains(handle, "@") {
			nh, hh := func(s string) (string, string) {
				if ar := strings.Split(s, "@"); len(ar) == 2 {
					return ar[0], ar[1]
				}
				return "", ""
			}(handle)

			handle = nh
			host = hh
		}
	case "https", "http":
		host = handle
	default:
		return renderApiError(c, http.StatusBadRequest, fmt.Errorf("unsupported resource type: %s", typ))
	}

	if host != wc.context.GetConfig().Host {
		return renderApiError(c, http.StatusNotFound, fmt.Errorf("resource not found %s", res))
	}

	user, err := model.GetUser(wc.context.GetDB(), handle)
	if err != nil || !user.ActivityPubEnabled() {
		return renderApiError(c, http.StatusNotFound, fmt.Errorf("resource not found %s", res))
	}

	resp := dto.WellKnownNode{
		Subject: res,
		Links: []dto.WellKnownLink{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: fmt.Sprintf("https://%s/ap/users/%s", wc.context.GetConfig().Host, user.Username),
			},
		},
	}

	if rels, hasRel := c.QueryParams()["rel"]; hasRel && len(rels) > 0 {
		allowed := make(map[string]struct{}, len(rels))
		for _, rel := range rels {
			allowed[rel] = struct{}{}
		}

		filtered := make([]dto.WellKnownLink, 0, len(resp.Links))
		for _, link := range resp.Links {
			if _, ok := allowed[link.Rel]; ok {
				filtered = append(filtered, link)
			}
		}
		resp.Links = filtered
	}

	c.Response().Header().Set(echo.HeaderContentType, "application/jrd+json")
	c.Response().WriteHeader(http.StatusOK)
	return json.NewEncoder(c.Response()).Encode(resp)
}

func (wc *wellKnownController) HostMeta(c echo.Context) error {
	host := wc.context.GetConfig().Host
	template := fmt.Sprintf("https://%s/.well-known/webfinger?resource={uri}", host)

	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0">
  <Link rel="lrdd" type="application/jrd+json" template="%s"/>
</XRD>
`, template)

	c.Response().Header().Set(echo.HeaderContentType, "application/xrd+xml")
	return c.String(http.StatusOK, body)
}

func splitResourceString(res string) (string, string) {
	split := ":"
	if strings.Contains(res, "://") {
		split = "://"
	}
	ar := strings.SplitN(res, split, 2)
	if len(ar) != 2 {
		return "", ""
	}
	typ := ar[0]
	handle := ar[1]
	if len(handle) == 0 {
		return "", ""
	}
	if handle[0] == '@' && len(handle) > 1 {
		handle = handle[1:]
	}
	return typ, handle
}
