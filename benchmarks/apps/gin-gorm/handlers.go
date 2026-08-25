package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
	"github.com/gombit-dev/gombit/contract"
	"github.com/gombit-dev/gombit/database"
	"gorm.io/gorm"
)

// Handler serves the canonical /api/projects routes (benchmarks/docs/schema.md)
// over plain Gin + GORM — this repo's primary framework-tax control: same
// language, runtime, ORM family, and response envelope as the Gombit app,
// without Huma/framework.App around it.
type Handler struct {
	DB *gorm.DB
}

// newRouter builds the route table shared by main() and the integration
// tests, so tests exercise exactly what the running server serves rather
// than a hand-rewired approximation of it.
func newRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	h := &Handler{DB: db}
	api := router.Group("/api")
	api.GET("/projects", h.list)
	api.GET("/projects/:id", h.get)
	api.POST("/projects", h.create)
	api.PATCH("/projects/:id", h.update)
	api.DELETE("/projects/:id", h.delete)
	router.GET("/livez", func(c *gin.Context) { c.Status(http.StatusOK) })

	return router
}

type createProjectRequest struct {
	OwnerID     uint   `json:"owner_id" binding:"required"`
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description"`
}

type updateProjectRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description"`
}

// list handles GET /api/projects?page=&limit=: a single page of projects,
// deterministically ordered (id DESC), owner preloaded in the same query
// set (one query for the page, one for its distinct owners: GORM's
// .Preload, not one owner query per row) so this endpoint never N+1s.
func (h *Handler) list(c *gin.Context) {
	page, limit := clampPage(queryInt(c, "page"), queryInt(c, "limit"))

	var total int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&Project{}).Count(&total).Error; err != nil {
		writeInternal(c, "count projects")
		return
	}

	var rows []Project
	err := h.DB.WithContext(c.Request.Context()).
		Preload("Owner").
		Order("id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		writeInternal(c, "list projects")
		return
	}

	items := make([]shared.ProjectData, 0, len(rows))
	for _, row := range rows {
		items = append(items, toProjectData(row))
	}

	c.JSON(http.StatusOK, contract.DataMeta[[]shared.ProjectData, shared.PageMeta]{
		Data: items,
		Meta: &shared.PageMeta{Page: page, Limit: limit, Total: total},
	})
}

// get handles GET /api/projects/:id.
func (h *Handler) get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeNotFound(c, "project not found")
		return
	}

	var row Project
	if err := h.DB.WithContext(c.Request.Context()).Preload("Owner").First(&row, uint(id)).Error; err != nil {
		writeMapped(c, database.MapLoadError(c.Request.Context(), err, "project not found", "load project"))
		return
	}

	c.JSON(http.StatusOK, contract.Data[shared.ProjectData]{Data: toProjectData(row)})
}

// create handles POST /api/projects.
func (h *Handler) create(c *gin.Context) {
	var body createProjectRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		writeMapped(c, contract.WithContext(c.Request.Context(), contract.Validation(err.Error(), nil)))
		return
	}

	row := Project{
		OwnerID:     body.OwnerID,
		Name:        body.Name,
		Description: body.Description,
	}
	if err := h.DB.WithContext(c.Request.Context()).Create(&row).Error; err != nil {
		writeMapped(c, database.MapPersistError(c.Request.Context(), err, "project already exists", "create project"))
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Preload("Owner").First(&row, row.ID).Error; err != nil {
		writeInternal(c, "reload created project")
		return
	}

	c.JSON(http.StatusCreated, contract.Data[shared.ProjectData]{Data: toProjectData(row)})
}

// update handles PATCH /api/projects/:id: load, apply the provided fields,
// save the full row — the same load-mutate-save shape the runtime admin's
// generic update handler uses (admin/resources.go).
func (h *Handler) update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeNotFound(c, "project not found")
		return
	}

	var body updateProjectRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		writeMapped(c, contract.WithContext(c.Request.Context(), contract.Validation(err.Error(), nil)))
		return
	}

	var row Project
	if err := h.DB.WithContext(c.Request.Context()).First(&row, uint(id)).Error; err != nil {
		writeMapped(c, database.MapLoadError(c.Request.Context(), err, "project not found", "load project"))
		return
	}

	if body.Name != nil {
		row.Name = *body.Name
	}
	if body.Description != nil {
		row.Description = *body.Description
	}
	if err := h.DB.WithContext(c.Request.Context()).Save(&row).Error; err != nil {
		writeMapped(c, database.MapPersistError(c.Request.Context(), err, "project already exists", "update project"))
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Preload("Owner").First(&row, row.ID).Error; err != nil {
		writeInternal(c, "reload updated project")
		return
	}

	c.JSON(http.StatusOK, contract.Data[shared.ProjectData]{Data: toProjectData(row)})
}

// delete handles DELETE /api/projects/:id.
func (h *Handler) delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeNotFound(c, "project not found")
		return
	}

	var row Project
	if err := h.DB.WithContext(c.Request.Context()).First(&row, uint(id)).Error; err != nil {
		writeMapped(c, database.MapLoadError(c.Request.Context(), err, "project not found", "load project"))
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Delete(&row).Error; err != nil {
		writeInternal(c, "delete project")
		return
	}

	c.JSON(http.StatusOK, contract.Data[map[string]bool]{Data: map[string]bool{"ok": true}})
}

func toProjectData(row Project) shared.ProjectData {
	return shared.ProjectData{
		ID:          row.ID,
		OwnerID:     row.OwnerID,
		OwnerName:   row.Owner.Name,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func queryInt(c *gin.Context, key string) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return 0
	}
	return value
}

func clampPage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// writeMapped renders a *contract.ErrorEnvelope (from database.MapLoadError /
// MapPersistError) as {"error": {...}} at its D10 status code. Gin has no
// built-in concept of these error types, so this is the one piece of glue
// idiomatic Gin code wouldn't otherwise need — kept minimal on purpose.
func writeMapped(c *gin.Context, err error) {
	envelope, ok := err.(*contract.ErrorEnvelope)
	if !ok {
		writeInternal(c, err.Error())
		return
	}
	c.JSON(envelope.GetStatus(), envelope)
}

func writeNotFound(c *gin.Context, message string) {
	writeMapped(c, contract.WithContext(c.Request.Context(), contract.NotFound(message)))
}

func writeInternal(c *gin.Context, message string) {
	writeMapped(c, contract.WithContext(c.Request.Context(), contract.Internal(message)))
}
