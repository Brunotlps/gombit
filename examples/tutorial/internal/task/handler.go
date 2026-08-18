package task

import (
	"context"
	"strconv"

	"github.com/LAA-Software-Engineering/gombit/contract"
	"gorm.io/gorm"
)

// Handler serves task HTTP operations over GORM. The generator emits a thin
// handler like this one; `--service` / `--repo` add the pass-through layers.
type Handler struct {
	DB *gorm.DB
}

type taskData struct {
	ID    uint   `json:"id" example:"1" doc:"Task identifier"`
	Title string `json:"title" doc:"Title"`
	Done  bool   `json:"done" doc:"Done"`
}

type listTasksOutput struct {
	Body contract.DataMeta[[]taskData, contract.PageMeta]
}

type getTaskInput struct {
	ID string `path:"id" doc:"Task identifier"`
}

type getTaskOutput struct {
	Body contract.Data[taskData]
}

type createTaskInput struct {
	Body struct {
		Title string `json:"title" minLength:"1" maxLength:"255" doc:"Title"`
		Done  bool   `json:"done" doc:"Done"`
	}
}

type createTaskOutput struct {
	Body contract.Data[taskData]
}

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listTasksOutput, error) {
	var rows []Task
	if err := h.DB.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("list tasks"))
	}
	items := make([]taskData, 0, len(rows))
	for _, row := range rows {
		items = append(items, toTaskData(row))
	}
	return &listTasksOutput{
		Body: contract.DataMeta[[]taskData, contract.PageMeta]{
			Data: items,
			Meta: &contract.PageMeta{Page: 1, PerPage: 20, Total: int64(len(items))},
		},
	}, nil
}

func (h *Handler) get(ctx context.Context, input *getTaskInput) (*getTaskOutput, error) {
	id, err := strconv.ParseUint(input.ID, 10, 64)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.NotFound("task not found"))
	}
	var row Task
	if err := h.DB.WithContext(ctx).First(&row, uint(id)).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.NotFound("task not found"))
	}
	return &getTaskOutput{
		Body: contract.Data[taskData]{Data: toTaskData(row)},
	}, nil
}

func (h *Handler) create(ctx context.Context, input *createTaskInput) (*createTaskOutput, error) {
	row := Task{
		Title: input.Body.Title,
		Done:  input.Body.Done,
	}
	if err := h.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("create task"))
	}
	return &createTaskOutput{
		Body: contract.Data[taskData]{Data: toTaskData(row)},
	}, nil
}

func toTaskData(row Task) taskData {
	return taskData{ID: row.ID, Title: row.Title, Done: row.Done}
}
