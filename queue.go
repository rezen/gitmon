package gitmon

import (
	"fmt"
	"github.com/gocraft/work"
)


type JobAndHandler interface {
	Name() string // For unique jobs
	Handle(*work.Job) error
	ToArgs() map[string]interface{}
}

type Queue struct {
	Handlers map[string]JobAndHandler
	Emitter *BetterBus
	Enqueuer *work.Enqueuer
}

func CreateQueueWithApp(app *App) *Queue {
	return &Queue {
		Emitter: app.Emitter,
		Handlers: make(map[string]JobAndHandler),
		Enqueuer: app.Enqueuer,
	}
}

func (q *Queue) AddHandler(handler JobAndHandler) *Queue {
	name := handler.Name()
	// @todo what if same name exists?
	q.Handlers[name] = handler
	return q
}

func (q *Queue) Push(j JobAndHandler) {
	fmt.Println(q.Enqueuer.Enqueue(j.Name(), j.ToArgs()))
	// q.Emitter.Publish("job.pushed")
}

func (q *Queue) WrappedHandler(name string) (func(*work.Job) error) {
	return func(job *work.Job) error {
		return nil
	}
}