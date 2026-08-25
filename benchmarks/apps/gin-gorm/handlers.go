package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Postgres SQLSTATE codes this app maps explicitly. See mapPersistError.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// Handler serves the canonical /api/projects routes (benchmarks/docs/schema.md)
// over plain Gin + GORM, this repo's primary framework-tax control. Same
// language, runtime, ORM family, and response envelope as the Gombit app,
// without Huma/framework.App around it, and without importing Gombit's
// contract/database packages either (contract.ErrorEnvelope is a
// huma.StatusError, so importing contract would pull Huma into this
// binary's dependency graph regardless of whether any handler here touches
// it). Response envelope types come from benchmarks/apps/shared instead —
// same D10 JSON shape, zero framework dependency.
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
// deterministically ordered (id DESC). 3 queries for a non-empty page —
// COUNT, page SELECT, one batched owner IN (...) via GORM's .Preload — not
// one owner query per row; see benchmarks/docs/schema.md "Canonical CRUD
// API" for the pinned shape TestListDoesNotN1 enforces.
func (h *Handler) list(c *gin.Context) {
	page, limit := clampPage(queryInt(c, "page"), queryInt(c, "limit"))

	var total int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&Project{}).Count(&total).Error; err != nil {
		writeError(c, shared.InternalError("count projects"))
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
		writeError(c, shared.InternalError("list projects"))
		return
	}

	items := make([]shared.ProjectData, 0, len(rows))
	for _, row := range rows {
		items = append(items, toProjectData(row))
	}

	c.JSON(http.StatusOK, shared.DataMeta[[]shared.ProjectData, shared.PageMeta]{
		Data: items,
		Meta: &shared.PageMeta{Page: page, Limit: limit, Total: total},
	})
}

// get handles GET /api/projects/:id.
func (h *Handler) get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, shared.NotFoundError("project not found"))
		return
	}

	var row Project
	if err := h.DB.WithContext(c.Request.Context()).Preload("Owner").First(&row, uint(id)).Error; err != nil {
		writeError(c, mapLoadError(err, "project not found", "load project"))
		return
	}

	c.JSON(http.StatusOK, shared.Data[shared.ProjectData]{Data: toProjectData(row)})
}

// create handles POST /api/projects.
func (h *Handler) create(c *gin.Context) {
	var body createProjectRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		writeError(c, shared.ValidationError(err.Error(), nil))
		return
	}

	row := Project{
		OwnerID:     body.OwnerID,
		Name:        body.Name,
		Description: body.Description,
	}
	if err := h.DB.WithContext(c.Request.Context()).Create(&row).Error; err != nil {
		writeError(c, mapPersistError(err, "project already exists", "create project"))
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Preload("Owner").First(&row, row.ID).Error; err != nil {
		writeError(c, shared.InternalError("reload created project"))
		return
	}

	c.JSON(http.StatusCreated, shared.Data[shared.ProjectData]{Data: toProjectData(row)})
}

// update handles PATCH /api/projects/:id: load, apply the provided fields,
// save the full row — the same load-mutate-save shape the runtime admin's
// generic update handler uses (admin/resources.go).
func (h *Handler) update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, shared.NotFoundError("project not found"))
		return
	}

	var body updateProjectRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		writeError(c, shared.ValidationError(err.Error(), nil))
		return
	}
	// binding's `omitempty` on a *string skips `max=255` when the pointed-to
	// value is empty, not just when the pointer is nil — a client that sends
	// {"name":""} (as opposed to omitting the field) would otherwise bypass
	// the same non-blank-name rule POST /api/projects enforces via
	// `required`. Checked explicitly since binding tags can't express "at
	// least one non-whitespace character, only when provided" on a pointer.
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		writeError(c, shared.ValidationError("name must not be blank", nil))
		return
	}

	var row Project
	if err := h.DB.WithContext(c.Request.Context()).First(&row, uint(id)).Error; err != nil {
		writeError(c, mapLoadError(err, "project not found", "load project"))
		return
	}

	if body.Name != nil {
		row.Name = *body.Name
	}
	if body.Description != nil {
		row.Description = *body.Description
	}
	if err := h.DB.WithContext(c.Request.Context()).Save(&row).Error; err != nil {
		writeError(c, mapPersistError(err, "project already exists", "update project"))
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Preload("Owner").First(&row, row.ID).Error; err != nil {
		writeError(c, shared.InternalError("reload updated project"))
		return
	}

	c.JSON(http.StatusOK, shared.Data[shared.ProjectData]{Data: toProjectData(row)})
}

// delete handles DELETE /api/projects/:id.
func (h *Handler) delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, shared.NotFoundError("project not found"))
		return
	}

	var row Project
	if err := h.DB.WithContext(c.Request.Context()).First(&row, uint(id)).Error; err != nil {
		writeError(c, mapLoadError(err, "project not found", "load project"))
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Delete(&row).Error; err != nil {
		writeError(c, shared.InternalError("delete project"))
		return
	}

	c.JSON(http.StatusOK, shared.Data[map[string]bool]{Data: map[string]bool{"ok": true}})
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

// mapLoadError maps a GORM read error to a D10 category error: record-not-
// found becomes not_found; anything else becomes internal. Local
// reimplementation of database.MapLoadError (github.com/gombit-dev/gombit/database)
// — same policy, no framework import.
func mapLoadError(err error, notFound, internal string) *shared.ErrorEnvelope {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shared.NotFoundError(notFound)
	}
	return shared.InternalError(internal)
}

// mapPersistError maps a GORM write error to a D10 category error by
// Postgres SQLSTATE: a unique-constraint violation becomes conflict; a
// foreign-key violation (e.g. POST /api/projects with an owner_id that
// doesn't reference an existing user) becomes validation_error — a bad
// client-supplied reference is invalid input, not a server failure, and
// issue #141 §15 requires every implementation reject equivalent invalid
// input the same way. Anything else becomes internal.
//
// database.MapPersistError (github.com/gombit-dev/gombit/database) only
// special-cases unique violations — reusing it here would have left FK
// violations falling through to internal (500), which is exactly the bug
// this function exists to not have. Projects have no unique business key of
// their own, so unique_violation can't currently be reached through this
// handler; the case is kept because a future field addition (e.g. a slug)
// could reach it, and disagreeing with the FK case only when a table
// happens to lack a unique constraint would be a much harder bug to notice.
func mapPersistError(err error, conflict, internal string) *shared.ErrorEnvelope {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return shared.ConflictError(conflict)
		case pgForeignKeyViolation:
			return shared.ValidationError("request references a resource that does not exist", nil)
		}
	}
	return shared.InternalError(internal)
}

func writeError(c *gin.Context, envelope *shared.ErrorEnvelope) {
	c.JSON(envelope.Status, envelope)
}
