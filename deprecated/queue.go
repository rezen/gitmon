package gitmon

import (	
	"time"
	"reflect"
	"github.com/satori/go.uuid"

	// "strconv"
	"encoding/json"
	"fmt"
	// "time"
	"github.com/jinzhu/gorm"
	// "github.com/google/uuid"
	// "github.com/borderstech/artifex"
	// "github.com/asaskevich/EventBus"
	"github.com/gocraft/work"

)

type Job struct {
	ID  uuid.UUID `gorm:"type:varchar(36);primary_key;"`	
	
	UserID int

	Class string 
	Name string
	Status int 
	Data string 
	Failed int
	CreatedAt time.Time
	UpdatedAt time.Time

	// Fields not saved to DB
	CreateToDone time.Duration `gorm:"-"`
	Error error `gorm:"-"`
	Handler SelfHandlingJob `gorm:"-"`
}

type SelfHandlingJob interface {
	Name() string // For unique jobs
	Handle() error
}

type JobAndHandler interface {
	Name() string // For unique jobs
	Handle(*work.Job) error
	ToArgs() map[string]interface{}
}


type Queue struct {
	DB *gorm.DB
	Handlers map[string]SelfHandlingJob
	Emitter *BetterBus
	// Stats statsd.Client?
}

func CreateQueueWithApp(app *App) *Queue {
	return &Queue {
		DB: app.DB,
		Emitter: app.Emitter,
		Handlers: make(map[string]SelfHandlingJob),
	}
}

func (q *Queue) ResolveClass(handler SelfHandlingJob) string {
	if t := reflect.TypeOf(handler); t.Kind() == reflect.Ptr {
        return t.Elem().Name()
    } else {
        return t.Name()
	}
}

func (q *Queue) AddHandler(handler SelfHandlingJob) {
	class := q.ResolveClass(handler)
	q.Handlers[class] = handler
}

func (q *Queue) Push(work SelfHandlingJob) {
	data, _ := json.Marshal(work)
	klass := q.ResolveClass(work)
	name := work.Name()
	job := &Job{
		Status: 0,
		UserID: 0,
		Class: klass,
		Name: name,
		Data: string(data),
	}

	if len(name) > 0 {
		count := 0
		q.DB.Table("jobs").Where("class = ? AND name = ? AND status <= 1", klass, name).Count(&count)

		if count > 0 {
			q.Emitter.Publish("job.already_queued", job)
			return
		}
	}
	job.CreatedAt = time.Now()
	job.UpdatedAt = job.CreatedAt
	q.DB.Save(job)
}

func (q *Queue) Handle(job *Job) error {
	job.Status = 1
	q.DB.Model(job).Update(map[string]interface{}{
		"status": job.Status,
		"updated_at": time.Now(),
	})
	// @todo save
	handler, ok := q.Handlers[job.Class]; 
	
	if !ok {
		// @todo delete job?
		fmt.Println("No handler for " + job.Class)
		return nil
	}
	err := json.Unmarshal([]byte(job.Data), handler)

	if err != nil {
		job.Error = err
		job.Status = 3
		q.DB.Model(job).Update(map[string]interface{}{
			"status": job.Status,
			"updated_at": time.Now(),
		})		
		return err
	}
	// @todo handler 
	err = handler.Handle()

	if err != nil {
		job.Error = err
		job.Status = 3

		q.DB.Model(job).Update(map[string]interface{}{
			"status": job.Status,
			"updated_at": time.Now(),
		})
		return err
	}

	job.Status = 9
	q.DB.Model(job).Update(map[string]interface{}{
		"status": job.Status,
		"updated_at": time.Now(),
	})
	return nil
}

func (q *Queue) Read() (jobs []Job) {
	q.DB.Where("status = ?", 0).Find(&jobs)
	return jobs
}

// BeforeCreate will set a UUID rather than numeric ID.
func (j *Job) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", uuid.NewV4())
}