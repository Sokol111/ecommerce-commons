package consumer

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type Worker interface {
	Start()
	Stop()
}

type baseWorker struct {
	name       string
	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	log        *zap.Logger
	runFunc    func(ctx context.Context)
}

func newBaseWorker(name string, log *zap.Logger, runFunc func(ctx context.Context)) Worker {
	return &baseWorker{
		name:    name,
		log:     log,
		runFunc: runFunc,
	}
}

func (w *baseWorker) Start() {
	w.log.Info("starting " + w.name)
	w.ctx, w.cancelFunc = context.WithCancel(context.Background())
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.runFunc(w.ctx)
		w.log.Info(w.name + " stopped")
	}()
}

func (w *baseWorker) Stop() {
	w.log.Info("stopping " + w.name)
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	w.wg.Wait()
}
