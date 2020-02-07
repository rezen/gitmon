package gitmon

import (
	// "encoding/json"
	_ "net/http/pprof"
	"encoding/base64"

	"strings"
	"strconv"
	"fmt"
	// https://www.thepolyglotdeveloper.com/2017/04/using-sqlite-database-golang-application/
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/go-playground/validator.v9"
	"net/http"
	"github.com/gocraft/work"


	"github.com/labstack/echo-contrib/session"
	"github.com/wader/gormstore"
	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/echoview"
	"github.com/biezhi/gorm-paginator/pagination"
	"html/template"

)

// User editable fields are tagged ui
// https://github.com/mailgun/mailgun-go
// https://github.com/PagerDuty/go-pagerduty


func Server() {
	fmt.Println("SERVER")
	app := CreateApp()	
	e := app.Server
	e.Validator = &AppValidator{validator: app.Validator}
	
	config := goview.DefaultConfig
	config.DisableCache = true
	config.Funcs["base64"] =  func(str []byte) template.HTML {
		return template.HTML(base64.StdEncoding.EncodeToString(str))
	}
	
	e.Renderer = echoview.New(config)

	// Middleware
	store := gormstore.New(app.DB, []byte("secret"))

	// e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	e.Use(session.MiddlewareWithConfig(session.Config{
		Store: store,
	}))
	e.Pre(middleware.MethodOverrideWithConfig(middleware.MethodOverrideConfig{
		Getter: middleware.MethodFromForm("_method"),
	}))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup: "form:csrf",
	  }))

	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            3600,
		// ContentSecurityPolicy: "default-src 'self'",
	}))


	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get("session", c)
			isAuthenticated := false
			if err != nil {
				isAuthenticated, _ = sess.Values["isAuthenticated"].(bool);
			}
			c.Set("isAuthenticated", isAuthenticated)
			return next(c)
		}
	})

	RegisterAuthRoutesWithApp(app)
	// Route => handler
	e.GET("/", func(c echo.Context) error {
		// https://github.com/gorilla/sessions/blob/master/sessions.go#L31

		return c.String(http.StatusOK, "Hello, World!\n")
	})

	admin := e.Group("/admin", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, _ := session.Get("session", c)
			if auth, ok := sess.Values["isAuthenticated"].(bool); !ok || !auth {
				return c.Redirect(302, "/auth/login")
			}
			userId := sess.Values["userId"].(int)
			user := app.Users.GetById(userId)
			user.IP = c.RealIP()
			user.UserAgent = c.Request().UserAgent()

			c.Set("sessionId", sess.ID)
			c.Set("isAuthenticated", true)
			c.Set("userId", userId)
			c.Set("user", user)
			return next(c)
		}
	})

	admin.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get("user").(*User)

			if len(user.OTPSecret) == 0 {
				return next(c)
			}
			sess, _ := session.Get("session", c)
			if verified, ok := sess.Values["otpVerified"].(bool); ok && verified {
				return next(c)
			}
		
			return c.Redirect(302, "/auth/login/otp") 
		}
	})

	admin.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get("user").(*User)
			method := c.Request().Method

			fmt.Println("CAN:", method, user.Email, "\n")

			// return c.String(http.StatusUnauthorized, "DENIED")
			return next(c)
		}
	})

	admin.GET("", func(c echo.Context) error { 
		return c.String(http.StatusOK, "admin")
	})

	admin.POST("/sites_bulk", func(c echo.Context) error {
		user := c.Get("user").(*User)
		urls := strings.Split(c.FormValue("urls"), "\n")
		added := 0
		fmt.Println(len(urls))
		for _, link := range  urls {
			link = strings.TrimSpace(link)
			site := &Site{
				Url: link,
				UserID: user.ID,
				IsEnabled: true,
			}
			err := app.Sites.Create(site)

			if err == nil {
				added = added + 1
			}
			fmt.Println("Added")
		}
		return c.Redirect(302, "/admin/sites?bulk_created=" + strconv.Itoa(added))
	})

	admin.POST("/sites", func(c echo.Context) error {
		user := c.Get("user").(*User)
		request := new(SiteRequest)
		if err := c.Bind(request); err != nil {
			panic(err)
		}

		if err := c.Validate(request); err != nil {
			for _, e := range err.(validator.ValidationErrors) {
				fmt.Println(e.Param(), e.Tag())
			}
			return c.Redirect(302, "/admin/sites?error=create")
		}

		site := &Site{
			Url: c.FormValue("url"),
			UserID: user.ID,
			IsEnabled: true,
			Tags: strings.Split(c.FormValue("tags"), ","),
		}
		err := app.Sites.Create(site)

		if err != nil {
			fmt.Println(err)
			return c.Redirect(302, "/admin/sites?error=create")
		}


		return c.Redirect(302, "/admin/sites?id=" + strconv.Itoa(site.ID))
	})

	admin.GET("/sites/:id/trigger", func(c echo.Context) error {
		id := c.Param("id")
		userId := c.Get("userId").(int)

		site := app.Sites.ById(id)
		if site.UserID != userId {
			return c.JSON(http.StatusOK, "FAIL")
		}
		app.Queue.Push(&ExecuteScanJob{Site: site, ScannerID: 1})
		app.Queue.Push(&ExecuteScanJob{Site: site, ScannerID: 2})

		return c.Redirect(302, "/admin/sites?triggered=" + id)
	})


	admin.DELETE("/sites/:id", func(c echo.Context) error {
		id := c.Param("id")
		userId := c.Get("userId").(int)

		site := app.Sites.ById(id)
		if site.UserID != userId {
			return c.JSON(http.StatusOK, "FAIL")
		}

		app.DB.Delete(site)
		app.DB.Exec("DELETE FROM scans where site_id = ?", site.ID)
		return c.Redirect(302, "/admin/sites")
	})

	admin.GET("/sites", func(c echo.Context) error {
		userId := c.Get("userId").(int)
		search := c.QueryParam("s")
		sites := []Site{}

		if len(search) > 0 {
			sites = app.Sites.SearchByUser(search, User{ID: userId})
		} else {
			sites = app.Sites.ByUser(User{ID: userId})
		}

		return c.Render(http.StatusOK, "sites", echo.Map{
			"title": "test",
			"sites": sites,
			"csrf": c.Get("csrf"),
		})	
	})

	admin.GET("/jobs", func(c echo.Context) error {
		qClient := work.NewClient(app.Namespace, app.Redis)
		queues, _ := qClient.Queues()
		pools, _ := qClient.WorkerPoolHeartbeats()
		observations, _ := qClient.WorkerObservations()


		return c.Render(http.StatusOK, "jobs", echo.Map{
			"title": "test",
			"queues": queues,
			"pools": pools,
			"observations": observations,
			"worker_is_alive": app.State.Has("worker_process"),
			"worker_process": app.State.Get("worker_process"),

			"worker_util": app.State.Get("worker_util"),
			"worker_heartbeat": app.State.Get("worker_heartbeat"),
			"csrf": c.Get("csrf"),
		})	
	})

	admin.GET("/users", func(c echo.Context) error {
		// Is user super?
		var users []User
		page, _ := strconv.Atoi(c.QueryParam("page"))
		limit, _ := strconv.Atoi(c.QueryParam("limit"))
		paginator := pagination.Paging(&pagination.Param{
			DB:      app.DB.Where("id > ?", 0),
			Page:    page,
			Limit:   limit,
			OrderBy: []string{"id desc"},
		}, &users)
		fmt.Println(paginator)
		return c.Render(http.StatusOK, "users", echo.Map{
			"title": "users",
			"users": users,
			"csrf": c.Get("csrf"),
		})	
	})

	admin.GET("/scans", func(c echo.Context) error {
		userId := c.Get("userId").(int)
		siteId := c.QueryParam("site")
		scannerId := c.QueryParam("scanner")

		where := map[string]interface{}{
			"user_id": userId, 
		}
	
		if len(siteId) > 0 {
			where["site_id"] = siteId
		} 

		if len(scannerId) > 0 {
			where["scanner_id"] = scannerId
		} 
	
		scans := app.Scans.Where(where)
		return c.Render(http.StatusOK, "scans", echo.Map{
			"title": "test",
			"scans": scans,
			"csrf": c.Get("csrf"),
		})		
	})


	admin.GET("/config", func(c echo.Context) error {
		user := c.Get("user").(*User)
		config := &NotificationConfig{}
		app.DB.Where("user_id = ?", user.ID).First(&config)

		return c.Render(http.StatusOK, "config", echo.Map{
			"title": "test",
			"config": config,
			"user": user,
			"csrf": c.Get("csrf"),
		})
	})

	admin.GET("/profile", func(c echo.Context) error {
		sessionId := c.Get("sessionId").(string)
		user := c.Get("user").(*User)
		sessions := []UserSession{}
		app.DB.Where("user_id = ?", user.ID).Find(&sessions)
		otp, _ := GenerateOTPProvision(user.Email)
		for i, _ := range sessions {
			sessions[i].IsCurrent = (sessions[i].SessionID == sessionId)
		}

		return c.Render(http.StatusOK, "profile", echo.Map{
			"title": "test",
			"sessions": sessions,
			"user": user,
			"otp": otp,
			"csrf": c.Get("csrf"),
		})
	})

	admin.POST("/notifications/config", func(c echo.Context) error {
		user := c.Get("user").(*User)
		config := &NotificationConfig{}
		app.DB.Where("user_id = ?", user.ID).First(&config)
		
		bound := new(NotificationConfig)
		if err := c.Bind(bound); err != nil {
			panic(err)
		}

		bound.IsSlackEnabled = c.FormValue("is_slack_enabled") == "1"
		bound.IsEmailEnabled = c.FormValue("is_email_enabled") == "1"
		bound.IsPagerDutyEnabled = c.FormValue("is_pagerduty_enabled") == "1"

		if config.ID > 0 {
			bound.ID = config.ID
		}
		bound.User = *user
		bound.UserID = user.ID
	
		app.DB.Save(&bound)
		return c.Redirect(302, "/admin/config?updated=1")
	})
	
	debug := e.Group("/debug", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// @todo ensure request is local internal
			fmt.Println("DEBUG", c.RealIP())
			return next(c)
		}
	})

	debug.GET("/pprof/*", echo.WrapHandler(http.DefaultServeMux))

	// Start server
	e.Logger.Fatal(e.Start("127.0.0.1:7000"))
	app.Close()
}
