package gitmon


import (
	"strconv"
	"fmt"
	"net/http"
	"github.com/labstack/echo"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"gopkg.in/go-playground/validator.v9"
 	// "github.com/asaskevich/EventBus"
)

type AuthHandler struct {
	Users *Users
	Validator *validator.Validate
	Emitter *BetterBus
}

func (h AuthHandler) PostOTP(c echo.Context) error {
	token := c.FormValue("token")
	sess, _ := session.Get("session", c)
	userId := sess.Values["userId"].(int)
	user := h.Users.GetById(userId)
	verified, _ := VerifyOTPToken(token, user.OTPSecret)
	h.Emitter.Publish("user.otp_verification_attempt")
	if verified {
		sess.Values["otpVerified"] = true
		sess.Save(c.Request(), c.Response())
		return c.Redirect(302, c.Request().Referer()) 
	}
	return c.Redirect(302, "/auth/login/otp?error=token") 
}


func (h AuthHandler) GetOTP(c echo.Context) error {
	sess, _ := session.Get("session", c)
	if auth, ok := sess.Values["isAuthenticated"].(bool); !ok || !auth {
		return c.Redirect(302, "/auth/login") 
	}

	return c.Render(http.StatusOK, "auth/login_otp", echo.Map{
		"title": "test",
		"csrf": c.Get("csrf"),
	})
}

func (h AuthHandler) GetLogin(c echo.Context) error {
	sess, _ := session.Get("session", c)
	if auth, ok := sess.Values["isAuthenticated"].(bool); ok && auth {
		return c.Redirect(302, "/admin")
	}
	return c.Render(http.StatusOK, "auth/login", echo.Map{
		"title": "test",
		"csrf": c.Get("csrf"),
	})
}

func (h AuthHandler) PostLogin(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	valid, user := h.Users.VerifyCredentials(email, password)
	h.Emitter.Publish("user.login_attempt")

	if valid {
		// For remembering last user agent
		user.UserAgent = c.Request().UserAgent()
		user.IP = c.RealIP()
		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   3600 * 4, // 4 Hours
			HttpOnly: true,
			// Secure: true,
		}
		
		sess.Values["isAuthenticated"] = true
		sess.Values["userId"] = user.ID
		sess.Save(c.Request(), c.Response())
		h.Emitter.Publish("user.logged_in", user, sess.ID)
		return c.Redirect(302, "/admin/sites")
	}
	h.Emitter.Publish("user.login_fail")
	return c.Redirect(302, "/auth/login?login_failed=1")
}

func (h AuthHandler) GetRegister(c echo.Context) error {
	sess, _ := session.Get("session", c)
	if auth, ok := sess.Values["isAuthenticated"].(bool); ok && auth {
		return c.Redirect(302, "/admin")
	}
	return c.Render(http.StatusOK, "auth/register", echo.Map{
		"title": "Register",
		"email": "an2dres5m@gmail.com",
		"csrf": c.Get("csrf"),
	})
}

func (h AuthHandler) GetResetWithToken(c echo.Context) error {
	token := c.Param("token")
	return c.Render(http.StatusOK, "auth/reset_password", echo.Map{
		"title": "Reset Password",
		"token": token,
		"is_valid_reset": h.Users.IsValidResetToken(token),
		"csrf": c.Get("csrf"),
	})
}


func (h AuthHandler) PostResetWithToken(c echo.Context) error {
	token := c.Param("token")
	password := c.FormValue("password")
	h.Emitter.Publish("user.reset_password_token")

	if len(password) < 10 {
		return c.Redirect(302, "/auth/reset_password/t/" + token + "?error=password")
	}

	if 	password != c.FormValue("password_confirm") {
		return c.Redirect(302, "/auth/reset_password/t/" + token  + "?error=password_confirm")
	}

	reset := PasswordReset {
		Token: token,
		Email: c.FormValue("email"),
		IP: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}
	err := h.Users.ResetPassword(reset, password )

	if err != nil {
		panic(err)
	}

	return c.Redirect(302, "/auth/login?reset=1")
}

func (h AuthHandler) PostReset(c echo.Context) error {
	// @todo counter
	user := h.Users.GetByEmail(c.FormValue("email"))

	if len(user.Email) == 0  {
		return c.Redirect(302, "/auth/login?reset=1")
	}

	err := h.Users.RequestResetPassword(user)

	if err != nil {
		panic(err)
	}
	return c.Redirect(302, "/auth/login?reset=1")
}

func (h AuthHandler) PostRegister(c echo.Context) error {
	h.Emitter.Publish("user.register_attempt")

	password := c.FormValue("password")

	if 	password != c.FormValue("password_confirm") {
		h.Emitter.Publish("user.register_fail")
		return c.Redirect(302, "/auth/register?error=password_confirm")
	}

	user := &User{}

	if err := c.Bind(user); err != nil {
		panic(err)
	}

	errs := h.Validator.Var(user.Email, "required,email")
	
	if errs != nil {
		h.Emitter.Publish("user.register_fail")
		return c.Redirect(302, "/auth/register?error=email")
	}

	if err := c.Validate(user); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			fmt.Println(e.Param(), e.Tag())
		}
		h.Emitter.Publish("user.register_fail")
		return c.Redirect(302, "/auth/register?error=password")
	}

	err := h.Users.Create(user)

	if err != nil {
		h.Emitter.Publish("user.register_fail")
		return c.Redirect(302, "/auth/register?error=password")
	}

	return c.Redirect(302, "/auth/login")
}

func (h AuthHandler) EmailVerify(c echo.Context) error {
	userId, _ := strconv.Atoi(c.Param("id"))
	user := h.Users.GetById(userId)
	
	if user.ID != userId {
		// Not same user?
	}

	hash := c.QueryParam("hash")
	if user.IsValidVerification(hash) {
		h.Users.MarkEmailVerified(user)
		return c.Redirect(302, "/auth/login?verified=1")
	}
	return c.Redirect(302, "/auth/login")
}

func (h AuthHandler) GetLogout(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		MaxAge:   -1,
		HttpOnly: true,
	}
	sess.Values["isAuthenticated"] = false
	sess.Save(c.Request(), c.Response())
	h.Emitter.Publish("user.logged_out", sess.ID)
	return c.Redirect(302, "/auth/login")
}

func (h AuthHandler) GetReset(c echo.Context) error {
	sess, _ := session.Get("session", c)
	if auth, ok := sess.Values["isAuthenticated"].(bool); ok && auth {
		return c.Redirect(302, "/admin")
	}
	return c.Render(http.StatusOK, "auth/reset", echo.Map{
		"title": "Register",
		"csrf": c.Get("csrf"),
	})
}

func RegisterAuthRoutesWithApp(app *App) {
	e := app.Server
	h := &AuthHandler {
		Users: app.Users,
		Validator: app.Validator,
		Emitter: app.Emitter,
	}

	// @todo rate limit auth endpoints

	e.GET("/auth/login", h.GetLogin)
	e.POST("/auth/login", h.PostLogin)
	e.POST("/auth/login/otp", h.PostOTP)
	e.GET("/auth/login/otp", h.GetOTP)

	e.GET("/auth/register", h.GetRegister)
	e.POST("/auth/register", h.PostRegister)

	e.POST("/auth/reset_password", h.PostReset)
	e.GET("/auth/reset_password", h.GetReset)

	e.GET("/auth/reset_password/t/:token", h.GetResetWithToken)
	e.POST("/auth/reset_password/t/:token", h.PostResetWithToken)
	
	e.GET("/auth/verify_email/:id", h.EmailVerify)
	
	e.GET("/auth/logout", h.GetLogout)
}

