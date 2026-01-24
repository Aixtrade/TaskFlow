package task

import "context"

type Repository interface {
	Save(ctx context.Context, task *Task) error
	FindByID(ctx context.Context, id string) (*Task, error)
	FindByStatus(ctx context.Context, status Status, limit int) ([]*Task, error)
	FindByType(ctx context.Context, taskType string, limit int) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter ListFilter) ([]*Task, int64, error)
}

type ListFilter struct {
	Status   []Status
	Type     []string
	Queue    string
	Offset   int
	Limit    int
	OrderBy  string
	OrderDir string
}

func NewListFilter() ListFilter {
	return ListFilter{
		Offset:   0,
		Limit:    20,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}
}

func (f *ListFilter) WithStatus(status ...Status) *ListFilter {
	f.Status = status
	return f
}

func (f *ListFilter) WithType(taskType ...string) *ListFilter {
	f.Type = taskType
	return f
}

func (f *ListFilter) WithQueue(queue string) *ListFilter {
	f.Queue = queue
	return f
}

func (f *ListFilter) WithPagination(offset, limit int) *ListFilter {
	f.Offset = offset
	f.Limit = limit
	return f
}

func (f *ListFilter) WithOrder(orderBy, orderDir string) *ListFilter {
	f.OrderBy = orderBy
	f.OrderDir = orderDir
	return f
}
