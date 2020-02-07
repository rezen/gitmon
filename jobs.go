package gitmon


import (
	"errors"
	"github.com/gocraft/work"
)
type MailPasswordResetUpdate struct {
	PasswordReset *PasswordReset `json:"password_reset"`
	Mailer Emailer `json:"-"`
}

func (s MailPasswordResetUpdate) Name() string {
    return "mail:notify_password_reset"
}

func (s MailPasswordResetUpdate) ToArgs() map[string]interface{} {
    return map[string]interface{}{
        "email": s.PasswordReset.Email,
        "userAgent": s.PasswordReset.UserAgent,
        "ip": s.PasswordReset.IP,
    }
}

func (s MailPasswordResetUpdate) Handle(job *work.Job) error {
    email := job.ArgString("email")
    userAgent := job.ArgString("userAgent")
    ip := job.ArgString("ip")
    content := "Your password was reset" + userAgent + " " + ip
    return s.Mailer.Send(email, "Your password was reset", content)
}


type MailPasswordResetToken struct {
	PasswordReset *PasswordReset `json:"password_reset"`
	Mailer Emailer `json:"-"`
}

func (s MailPasswordResetToken) Name() string {
    return "mail:password_reset_token"
}

func (s MailPasswordResetToken) ToArgs() map[string]interface{} {
    return map[string]interface{}{
        "email": s.PasswordReset.Email,
        "token": s.PasswordReset.Token,
    }
}

func (s MailPasswordResetToken) Handle(job *work.Job) error {
    email := job.ArgString("email")
    token := job.ArgString("token")
    
    if len(token) == 0 {
		return errors.New("Invalid token")
    }

    content := `You have 10 minutes to reset your password via /auth/reset_password/t/` + token
    return s.Mailer.Send(email, "Reset your password", content)
}


type MailEmailVerification struct {
	User *User `json:"user"`
	Mailer Emailer `json:"-"`
}

func (s MailEmailVerification) Name() string {
    return "mail:email_verification"
}

func (s MailEmailVerification) ToArgs() map[string]interface{} {
    return map[string]interface{}{
        "email": s.User.Email,
    }
}

func (s MailEmailVerification) Handle(job *work.Job) error {
    email := job.ArgString("email")
    
    if len(email) == 0 {
		return errors.New("Invalid email")
    }

    return s.Mailer.Send(email, "Verify Your Email", `Howdy there! Thanks for registering at ..`)
}


type ExecuteScanJob struct {
	Site *Site `json:"site"`
	ScannerID int `json:"scanner_id"`
	ScanEngine *ScanEngine `json:"-"`
}

func (s ExecuteScanJob) Name() string {
	return "scan:site"
}

func (s ExecuteScanJob) ToArgs() map[string]interface{} {
    return map[string]interface{}{
        "siteId": s.Site.ID, 
        "siteUrl": s.Site.Url,
        "scanner": s.ScannerID,
    }
}

func (s ExecuteScanJob) Handle(job *work.Job) error {
	site := &Site{
		ID: int(job.ArgInt64("siteId")), 
		Url: job.ArgString("siteUrl"),
	}
	scannerId := job.ArgInt64("scanner")
	return s.ScanEngine.Scan(site, int(scannerId))
}
