package gitmon

import (
	"os"
	"fmt"
	"os/signal"
	"syscall"
	"time"
	"github.com/gocraft/work"
)

func DoEvery(run func(), interval time.Duration) (chan bool) {
	q := make(chan bool)
	ticker := time.NewTicker(interval)
	go func() {
		for true {
			select {
			case <- ticker.C:
				run()
			case <- q:
				fmt.Println("STOP-DO_EVERY")
				ticker.Stop()
				return
			}
		}
	}()
	return q
}

func RunInBackgroundLoop(run func(chan bool)) (chan bool) {
	q := make(chan bool)
	go func() {
		for true {
			select {
			case <- q:
				return
			default:
				run(q)
			}
		}
	}()
	return q
}

type Context struct{
    customerID int64
}

func (c *Context) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	fmt.Println("Starting job: ", job.Name)
	return next()
}

func (c *Context) FindCustomer(job *work.Job, next work.NextMiddlewareFunc) error {
	// If there's a customer_id param, set it in the context for future middleware and handlers to use.
	if _, ok := job.Args["customer_id"]; ok {
		c.customerID = job.ArgInt64("customer_id")
		if err := job.ArgError(); err != nil {
			return err
		}
	}

	return next()
}

func Worker() {
	// @todo get lock to see if only worker
	app := CreateApp()
	client := app.Stats
	allDone := make(chan bool)
	numberOfWorkers := 200

	pool := work.NewWorkerPool(Context{}, uint(numberOfWorkers), app.Namespace, app.Redis)

	// Add middleware that will be executed for each job
	pool.Middleware((*Context).Log)
	pool.Middleware((*Context).FindCustomer)

	for _, handler := range app.Queue.Handlers {
		pool.Job(handler.Name(), handler.Handle)
	}

	quitHeartbeat := DoEvery(func() {
		fmt.Println("---- HEARTBEAT ----")
		app.State.Put("worker_heartbeat", time.Now().String())	
	}, 20 * time.Second)
	

	quitCreateWork := DoEvery(func() {
		fmt.Println("---- CREATE_WORK ----")
		
		due := app.ScanEngine.ScansDue()
		fmt.Println(" ---- SCANS DUE",  len(due))
		for _, scan := range due {
			app.Queue.Push(&ExecuteScanJob{
				Site: &scan.Site, 
				ScannerID: scan.ScannerID,
			})
		}
		app.DB.Exec(`DELETE FROM user_sessions WHERE session_id in (select id from sessions where expires_at < NOW())`)
	}, 90 * time.Second)

	/*
	2800 yr 
	284,399.39
	vs 4000/yr
	*/
	quitStats := DoEvery(func() {
		fmt.Println("---- STATS ----")
		stats := GetMonitorStats()
		fmt.Println(" ", stats)
		client.Gauge("runtime_num_gc", int64(stats.NumGC))
		client.Gauge("runtime_live_objects", int64(stats.LiveObjects))
		client.Gauge("runtime_num_goroutine", int64(stats.NumGoroutine))
		client.Gauge("runtime_alloc", int64(stats.Alloc))
		client.Gauge("runtime_mallocs", int64(stats.Mallocs))
		client.Gauge("runtime_sys", int64(stats.Sys))
		// app.State.Put("worker_monitor_stats", stats)	
	}, 10 * time.Second)



	pool.Start()

	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		app.State.Delete("worker_process")
		fmt.Println("Close it all")
		quitHeartbeat <- true
		quitStats <- true
		quitCreateWork <- true
		fmt.Println("============ ON_CNTRL_C")
		allDone <- true
	}()

	details := GetRunningDetails()
	app.State.Put("worker_process", details.String())
	fmt.Println(details.String())
		
	<-allDone
	pool.Stop()
	client.Close()
}