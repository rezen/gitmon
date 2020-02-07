package gitmon

import (
	"os"
	"fmt"
	"os/signal"

	"syscall"
	"strconv"
	"time"
	statsd "github.com/smira/go-statsd"
)

// https://github.com/gocraft/work/blob/master/worker.go#L103
type Workit struct {
	Number int
	Queue *Queue
	Tasks chan Job
	Done chan WorkDone
	// 	Done chan bool
	Quit chan bool
	// AckQuit chan bool
	Stats *statsd.Client
}

type WorkDone struct {
	Error error
	JobClass string
}

func (w Workit) Stop() {
	fmt.Println(" - STOP_TRIGGERED", w.Number)
	go func() {
		w.Quit <- true
	}()
	// time.Sleep(time.Millisecond * 100)
	// fmt.Println("AFTER?!",  w.Number)
}

func (w Workit) Work() {
	go func() {
		defer func(){
			fmt.Println(" CLEAN UP WORKER FOR STOP", w.Number)
		}()
	fmt.Println("- STARTED WORKER", w.Number)
	loop:
		for true {
			select {
			case job := <-w.Tasks:
				err := w.Queue.Handle(&job)
				w.Stats.Incr("work.job_handled", 1)

				if err != nil {
					w.Stats.Incr("work.job_error", 1)
				}
		
				w.Done <- WorkDone{err, job.Class}
			case <-w.Quit:
				fmt.Println("--- QUIT-WORKER-HALTED", w.Number)
				break loop
			}
		}
	}()
}


// @todo delete jobs older than x
type Factory struct {
	NumberOfWorkers int
	Workers []Workit
	Quit chan bool
	Done chan WorkDone
	Jobs chan Job
	Queue *Queue
	State *State
	Stats *statsd.Client
}

func (f *Factory) SetupWorkers() {
	fmt.Println(" Setting up workers", f.NumberOfWorkers)
	for i := 0; i < f.NumberOfWorkers; i++ {
		f.Workers = append(f.Workers, Workit{
			Number: i,
			Queue: f.Queue,
			Tasks: f.Jobs,
			Done: f.Done,
			Quit: make(chan bool),
			// AckQuit: make(chan bool),
			Stats: f.Stats,
		})
	}
}

func (f *Factory) Work() {
	fmt.Println("Started work for ", len(f.Workers))
	for _, worker := range f.Workers {
		worker.Work()
	}
}

func (f *Factory) Stop() {
	fmt.Println("STOPPPPED work for ", len( f.Workers))
	count := len( f.Workers)
	stopWorkers := make(chan bool)
	for i, work := range f.Workers {
		fmt.Println(" >> STOP_WORKER_SEND", i)
		go func() {
			work.Stop()
			stopWorkers <- true
		}()
	}

	for i := 0; i < count; i++ {
		fmt.Println("<<< STOP_WORKER_DONE", i)
		<-stopWorkers
	}
}

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



// https://geeks.uniplaces.com/building-a-worker-pool-in-golang-1e6c0fdfd78c
func Worker() {


	// @todo get lock to see if only worker
	app := CreateApp()
	client := app.Stats
	allDone := make(chan bool)
	jobs := make(chan Job, 1000)
	done := make(chan WorkDone)
	numberOfWorkers := 200
	quit := make(chan bool)

	factory := &Factory{
		NumberOfWorkers: numberOfWorkers,
		Workers: []Workit{},
		Quit: quit,
		Jobs: jobs,
		Done: done,
		Queue: app.Queue,
		State: app.State,
		Stats: app.Stats,
	}

	quitHeartbeat := DoEvery(func() {
		for i := 1; i <= 1000; i++ {
			app.Queue.Push(&IncreaseCounter{})
		}
	}, 100 * time.Millisecond)



	/*
	quitHeartbeat := DoEvery(func() {
		fmt.Println("---- HEARTBEAT ----")
		app.State.Put("worker_heartbeat", time.Now().String())	
	}, 20 * time.Second)
	*/

	cleanupHeartbeat := DoEvery(func() {
		fmt.Println("CLEAN JOBS!")
		past := time.Now().Add(time.Duration(-5) * time.Minute)
		count := 0 
		app.DB.Model(&Job{}).Where("status = ? AND created_at <= ?", 9, past).Count(&count)
		app.DB.Where("status = ? AND created_at <= ?", 9, past).Delete(&Job{})
		fmt.Println("JOBS to CLEANUP", count)
		app.State.Put("last_job_cleanup", past.String())
		app.State.Put("last_job_cleanup_count", strconv.Itoa(count))

	}, 20 * time.Second)

	quitCreateWork := DoEvery(func() {
		fmt.Println("---- CREATE_WORK ----")
		/*
		due := app.ScanEngine.ScansDue()
		fmt.Println(" - SCANS DUE",  len(due))
		for _, scan := range due {
			app.Queue.Push(&ExecuteScanJob{
				Site: &scan.Site, 
				ScannerID: scan.ScannerID,
			})
		}*/
	}, 30 * time.Second)

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

	quit2 := RunInBackgroundLoop(func(q chan bool) {
		select {
		case unit := <-done:
			fmt.Println("  DONE", unit.JobClass)
		case <-q:
			fmt.Println("STOP DONE HANDLING!", len(done))
			return
		}
	})

	factory.SetupWorkers()
	factory.Work()

	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	DoEvery(func() {
		fmt.Println("")
		fmt.Println("=============== NUMBER OF OPSSSS", ops)
		fmt.Println("")

		c <- os.Interrupt
	}, 400 * time.Second)

	go func() {
		<-c
		app.State.Delete("worker_process")
		fmt.Println("Close it all")
		quitHeartbeat <- true
		quitStats <- true
		quitCreateWork <- true
		cleanupHeartbeat <- true
		quit <- true
		quit2 <- true
		fmt.Println("============ ON_CNTRL_C")
		allDone <- true
	}()

	details := GetRunningDetails()
	app.State.Put("worker_process", details.String())
	fmt.Println(details.String())
	
	// Run in background to feed workers
	go func() {
		for true {
			select {
			case <-quit:
				return
			default:
				todo := app.Queue.Read()
				

				for _, job := range todo {
					select {
						case <-quit:
							return
						default:
							client.Incr("work.job_pushed", 1)
							jobs <- job
					}
				}
				fmt.Println("Catching breath")
				// Catch your breath ...
				time.Sleep(time.Millisecond * 200)
			}
		}
	}()
	
	<-allDone
	// Tell all the workers to stop
	fmt.Println("--- JOBS LEFT ---", len(jobs))
	factory.Stop()
	fmt.Println("EXIT_LOOP")
	
	client.Close()
}