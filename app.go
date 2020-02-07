package gitmon

import (

	"time"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo"
	"github.com/asaskevich/EventBus"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	 _ "github.com/jinzhu/gorm/dialects/mysql"

	"gopkg.in/go-playground/validator.v9"
	statsd "github.com/smira/go-statsd"
	"github.com/gomodule/redigo/redis"
	"github.com/gocraft/work"


)

type AppValidator struct {
	validator *validator.Validate
}

func (v *AppValidator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}


type App struct {
	Version string
	Namespace string
	Scans *Scans
	Users *Users
	Sites *Sites
	State *State
	JobHandlers []JobAndHandler
	ScanEngine *ScanEngine
	DB *gorm.DB
	// Slack & pagerduty 
	Stats *statsd.Client
	

	Emitter *BetterBus
	Queue *Queue
	Validator *validator.Validate

	Server *echo.Echo
	Mail Emailer
	Redis *redis.Pool
	Enqueuer *	work.Enqueuer

	// Logger
	// AppToken
}

func (a *App) Close() {
	a.DB.Close()
	fmt.Println("CLOOSSSE")
}

func CreateApp() *App {
	app := &App{Version: "1.0", Namespace: "gitmon.prod"}

	// db, err := gorm.Open("sqlite3", "test.db")
	db, err := gorm.Open("mysql", "root:password123@tcp(127.0.0.1:3308)/devdb?parseTime=True&loc=Local&charset=utf8mb4&collation=utf8mb4_unicode_ci")

	if err != nil {
		panic(err)
	}
	// db.LogMode(true)

	app.Redis = &redis.Pool{
		MaxActive: 20,
		MaxIdle: 20,
		Wait: true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "127.0.0.1:6379")
		},
	}

	app.Enqueuer = work.NewEnqueuer(app.Namespace, app.Redis)


	SetupDatabase(db)

	app.Stats = statsd.NewClient("localhost:8120",statsd.MaxPacketSize(1400),statsd.MetricPrefix("web."), statsd.FlushInterval(1000*time.Millisecond))

	bus := EventBus.New()
	app.DB = db
	app.Validator = validator.New()
	app.Mail = CreateEmailer()
	app.Emitter = &BetterBus{bus}

	// https://github.com/foolin/goview/blob/master/view.go
	// RenderWriter

	app.Scans = &Scans{app.DB, app.Emitter}
	app.Users = &Users{app.DB, app.Emitter}
	app.Sites = &Sites{app.DB, app.Emitter}
	app.State = &State{app.DB, app.Emitter}
	app.ScanEngine = CreateScanEngineFromApp(app)

	app.ScanEngine.AddScanner(GitScanner())
	app.ScanEngine.AddScanner(ScannerIsAlive())

	app.Server = echo.New()
	app.Queue = CreateQueueWithApp(app)

	app.Queue.AddHandler(&ExecuteScanJob {
		ScanEngine: app.ScanEngine,
	}).AddHandler(&MailPasswordResetToken {
		Mailer: &FakeEmailer{},
	}).AddHandler(&MailPasswordResetUpdate {
		Mailer: &FakeEmailer{},
	}).AddHandler(&MailEmailVerification {
		Mailer: &FakeEmailer{},
	})

	app.Emitter.SubscribeAsync("*", func(topic string, args ...interface{}) {
		fmt.Println("WIld?", topic, len(args))
	}, false)

	app.Emitter.SubscribeAsync("user.*", func(suffix string, args ...interface{}) {
		app.Stats.Incr("user." + suffix, 1)
	}, false)

	app.Emitter.Subscribe("user.created", func(user *User) {
		app.Queue.Push(&MailEmailVerification{User: user})
	})

	app.Emitter.SubscribeAsync("job.*", func(suffix string, args ...interface{}) {
		app.Stats.Incr("job." + suffix, 1)
	}, true)

	app.Emitter.Subscribe("site.created", func(site *Site) {
		fmt.Println("start")

		app.Queue.Push(&ExecuteScanJob{Site: site, ScannerID: 1})
		app.Queue.Push(&ExecuteScanJob{Site: site, ScannerID: 2})
		fmt.Println("end")
	})

	app.Emitter.Subscribe("user.request_password_reset", func(reset *PasswordReset) {
		// jobName string, args map[string]interface{}
		app.Queue.Push(&MailPasswordResetToken {
			PasswordReset: reset,
		})
	})

	app.Emitter.Subscribe("user.password_reset", func(reset *PasswordReset) {
		app.Queue.Push(&MailPasswordResetUpdate{PasswordReset: reset})
	})

	app.Emitter.Subscribe("user.logged_in", func(user *User, sessionId string) {
		app.Users.LoggedIn(user)
		app.DB.Create(&UserSession{
			IP: user.IP,
			UserAgent: user.UserAgent,
			UserID: user.ID,
			SessionID: sessionId,
		})
	})

	app.Emitter.Subscribe("user.logged_out", func(sessionId string) {
		app.DB.Where("session_id <= ?", sessionId).Delete(&UserSession{})
	})

	return app
}