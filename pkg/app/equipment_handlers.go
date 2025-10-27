package app

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
	"github.com/stackus/hxgo/hxecho"
)

func (a *App) addRoutesEquipment(e *echo.Group) {
	equipmentGroup := e.Group("/equipment")
	equipmentGroup.POST("", a.addEquipment).Name = "equipment-create"
	equipmentGroup.POST("/:id", a.equipmentUpdateHandler).Name = "equipment-update"
	equipmentGroup.POST("/:id/delete", a.equipmentDeleteHandler).Name = "equipment-delete"
}

func (a *App) addEquipment(c echo.Context) error {
	u := a.getCurrentUser(c)
	p := database.Equipment{}

	if err := c.Bind(&p); err != nil {
		return a.redirectWithError(c, a.echo.Reverse("add-equipment"), err)
	}

	p.UserID = u.ID

	if err := p.Save(a.db); err != nil {
		return a.redirectWithError(c, a.echo.Reverse("add-equipment"), err)
	}

	return c.Redirect(http.StatusFound, a.echo.Reverse("equipment"))
}

func (a *App) equipmentDeleteHandler(c echo.Context) error {
	e, err := a.getEquipment(c)
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("equipment-show", c.Param("id")), err)
	}

	if err := e.Delete(a.db); err != nil {
		return a.redirectWithError(c, a.echo.Reverse("equipment-show", c.Param("id")), err)
	}

	a.addNoticeT(c, "translation.The_equipment_s_has_been_deleted", e.Name)

	if hxecho.IsHtmx(c) {
		c.Response().Header().Set("Hx-Redirect", a.echo.Reverse("equipment"))
		return c.String(http.StatusFound, "ok")
	}

	return c.Redirect(http.StatusFound, a.echo.Reverse("equipment"))
}

func (a *App) equipmentUpdateHandler(c echo.Context) error {
	e, err := a.getEquipment(c)
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("equipment-edit", c.Param("id")), err)
	}

	e.DefaultFor = nil
	e.Active = (c.FormValue("active") == "true")

	if err := c.Bind(e); err != nil {
		return a.redirectWithError(c, a.echo.Reverse("equipment-edit", c.Param("id")), err)
	}

	if err := e.Save(a.db); err != nil {
		return a.redirectWithError(c, a.echo.Reverse("equipment-edit", c.Param("id")), err)
	}

	a.addNoticeT(c, "translation.The_equipment_s_has_been_updated", e.Name)

	return c.Redirect(http.StatusFound, a.echo.Reverse("equipment-show", c.Param("id")))
}
