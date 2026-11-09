package trip

import (
	"context"

	"github.com/midavuha1/go-course-lab/internal/infra/postgres/transaction"
)

type executorProvider interface {
	Executor(ctx context.Context) transaction.Executor
}
