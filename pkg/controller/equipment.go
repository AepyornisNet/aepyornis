package controller

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

type EquipmentController interface {
	GetEquipmentList(c echo.Context) error
	GetEquipment(c echo.Context) error
	CreateEquipment(c echo.Context) error
	UpdateEquipment(c echo.Context) error
	DeleteEquipment(c echo.Context) error
}

type equipmentController struct {
	context *container.Container
}

func NewEquipmentController(c *container.Container) EquipmentController {
	return &equipmentController{context: c}
}

func (ec *equipmentController) getEquipment(c echo.Context) (*model.Equipment, error) {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}

	e, err := ec.context.GetUser(c).GetEquipment(ec.context.GetDB(), id)
	if err != nil {
		return nil, err
	}

	return e, nil
}

// GetEquipmentList returns a paginated list of equipment for the current user
// @Summary      List equipment
// @Tags         equipment
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Param        page      query  int false "Page"
// @Param        per_page  query  int false "Items per page"
// @Success      200  {object}  api.PaginatedResponse[dto.EquipmentResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /equipment [get]
func (ec *equipmentController) GetEquipmentList(c echo.Context) error {
	user := ec.context.GetUser(c)

	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	var totalCount int64
	if err := ec.context.GetDB().Model(&model.Equipment{}).Where(&model.Equipment{UserID: user.ID}).Count(&totalCount).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	var equipment []*model.Equipment
	db := ec.context.GetDB().Where(&model.Equipment{UserID: user.ID}).
		Order("name DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&equipment).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewEquipmentListResponse(equipment)

	resp := dto.PaginatedResponse[dto.EquipmentResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetEquipment returns a single equipment by ID
// @Summary      Get equipment
// @Tags         equipment
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Equipment ID"
// @Produce      json
// @Success      200  {object}  api.Response[dto.EquipmentResponse]
// @Failure      404  {object}  api.Response[any]
// @Router       /equipment/{id} [get]
func (ec *equipmentController) GetEquipment(c echo.Context) error {
	e, err := ec.getEquipment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	resp := dto.Response[dto.EquipmentResponse]{
		Results: dto.NewEquipmentDetailResponse(e),
	}

	return c.JSON(http.StatusOK, resp)
}

// CreateEquipment creates a new equipment
// @Summary      Create equipment
// @Tags         equipment
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      201  {object}  api.Response[dto.EquipmentResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /equipment [post]
func (ec *equipmentController) CreateEquipment(c echo.Context) error {
	user := ec.context.GetUser(c)

	var e model.Equipment
	if err := c.Bind(&e); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	e.UserID = user.ID

	if err := e.Save(ec.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.EquipmentResponse]{
		Results: dto.NewEquipmentResponse(&e),
	}

	return c.JSON(http.StatusCreated, resp)
}

// UpdateEquipment updates an existing equipment
// @Summary      Update equipment
// @Tags         equipment
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Equipment ID"
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[dto.EquipmentResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      403  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /equipment/{id} [put]
func (ec *equipmentController) UpdateEquipment(c echo.Context) error {
	user := ec.context.GetUser(c)

	e, err := ec.getEquipment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	e.DefaultFor = nil

	if e.UserID != user.ID {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	if err := c.Bind(e); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	e.UserID = user.ID

	if err := e.Save(ec.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.EquipmentResponse]{
		Results: dto.NewEquipmentResponse(e),
	}

	return c.JSON(http.StatusOK, resp)
}

// DeleteEquipment deletes an equipment
// @Summary      Delete equipment
// @Tags         equipment
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Equipment ID"
// @Success      204  "Deleted"
// @Failure      403  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /equipment/{id} [delete]
func (ec *equipmentController) DeleteEquipment(c echo.Context) error {
	user := ec.context.GetUser(c)

	e, err := ec.getEquipment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if e.UserID != user.ID {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	if err := e.Delete(ec.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusNoContent)
}
