package work

import (
	libapp "github.com/EscanBE/go-lib/app"
	"github.com/EscanBE/go-lib/logging"
	workertypes "github.com/bcdevtools/validator-health-check/work/types"
)

// Worker represents for a worker, itself holds things needed for doing business logic, especially its own `WorkerContext`
type Worker struct {
	Id     string
	Ctx    *workertypes.WorkerContext
	logger logging.Logger
}

// NewWorker creates new worker and inject needed information
func NewWorker(wCtx *workertypes.WorkerContext) Worker {
	return Worker{
		Id:     wCtx.WorkerID,
		Ctx:    wCtx,
		logger: wCtx.AppCtx.Logger,
	}
}

// Start performs business logic of worker
func (w Worker) Start() {
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(w.logger)

	// TODO implement worker's business logic here
}
