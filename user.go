package gitmon

import (
	"time"
	"fmt"
	"io"
	"errors"
	"crypto/sha1"
	"encoding/hex"

	"github.com/jinzhu/gorm"
	// "github.com/asaskevich/EventBus"
	"golang.org/x/crypto/bcrypt"
	"crypto/subtle"
	"github.com/satori/go.uuid"
	// "github.com/avct/uasurfer"
	"github.com/mssola/user_agent"

)


type User struct {
	ID   int
	Name string `json:"name"`
	Email string `json:"email" validate:"required,email" gorm:"unique;not null"`
	LastLoggedInAt *time.Time `json:"-" gorm:"default:null"`
	LastUserAgent string `json:"-"`
	LastIP string `json:"-"`
	Password string `gorm:"-" json:"-"`
	PasswordHash string `json:"-" gorm:"not null"`
	APIKey string
	OTPSecret string
	EmailVerifiedAt *time.Time `gorm:"default:null"`
	CreatedAt time.Time

	// When a user is logged in, these details are used
	IP string `json:"-" gorm:"-"`
	UserAgent string `json:"-" gorm:"-"`
}


func (u User) EmailVerificationCode() []byte {
	h := sha1.New()
	io.WriteString(h, u.Email)
	sum := h.Sum(nil)
	dst :=  make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(dst, sum)
	return dst
}

func (u User) IsValidVerification(code string) bool {
	compare := u.EmailVerificationCode()
	return subtle.ConstantTimeCompare([]byte(code), compare) == 1
}

func HashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}
 
func (u *User) SetPassword(password string) error {
	if len(password) < 10 {
		return errors.New("Password too short")
	}
	u.PasswordHash = HashPassword(password)
	return nil
}

func (u *User) BeforeUpdate(scope *gorm.Scope) error {
	if len(u.Password) > 0 && len(u.PasswordHash) == 0 {
		scope.SetColumn("PasswordHash", HashPassword(u.Password))	
	}
	return nil
}

func (u *User) BeforeCreate(scope *gorm.Scope) error {
	if len(u.Password) > 0 && len(u.PasswordHash) == 0 {
		scope.SetColumn("PasswordHash", HashPassword(u.Password))	
	}
	return nil
}

type PasswordReset struct {
	Token string `gorm:"primary_key;unique;not null;"`
	User User `json:"-" gorm:"association_autoupdate:false"`
	Email string `json:"email" validate:"required,email" gorm:"not null"`
	UserAgent string `gorm:"-"`
	IP string `gorm:"-"`

	CreatedAt time.Time
}

type Users struct {
	DB *gorm.DB
	Emitter *BetterBus
}

func (u *Users) IsValidResetToken(token string) bool {
	// Tokens expire 10 minuts after creation
	then := time.Now().Add(time.Duration(-10) * time.Minute)
	count := 0
	u.DB.Where("token = ? AND created_at > ?", token, then).First(&PasswordReset{}).Count(&count)
	fmt.Println(token, count)
	return count > 0
}

func (u *Users) IsValidReset(email, token string) bool {
	reset := &PasswordReset{
		Email: email,
		Token: token,
	}
	// Tokens expire 10 minuts after creation
	then := time.Now().Add(time.Duration(-10) * time.Minute)
	count := 0
	u.DB.Where("email = ? AND token = ? AND created_at > ?", email, token, then).First(&reset).Count(&count)
	return count > 0
}

func (u *Users) MarkEmailVerified(user *User)  {
	u.DB.Model(user).Update(map[string]interface{}{
		"email_verified_at": time.Now(),
	})
}


func (u *Users) ResetPassword(reset PasswordReset, password string) error{
	if !u.IsValidReset(reset.Email, reset.Token) {
		return errors.New("Invalid or expired reset token")
	}

	user := u.GetByEmail(reset.Email)
	err := user.SetPassword(password)
	if err != nil {
		return err
	}
	u.DB.Save(user)
	u.DB.Delete(reset)
	u.Emitter.Publish("user.password_reset", &reset)
	return nil
}


func (u *Users) RequestResetPassword(user *User) error {
	token, err := GenerateRandomStringURLSafe(32)
	reset := &PasswordReset{
		User: *user,
		Token: token,
		Email: user.Email,
		CreatedAt: time.Now(),
	}

	if err != nil {
		return err
	}

	fmt.Println(reset.Token)
	u.DB.Delete(PasswordReset{}, "email = ?", user.Email)
	u.DB.Create(reset)
	u.Emitter.Publish("user.request_password_reset", reset)
	return nil
}

func (u *Users) EmailExists(email string) bool  {
	count := 0
	u.DB.Where("email = ?", email).First(&User{}).Count(&count)
	return count > 0
}

func (u *Users) GetByEmail(email string) *User  {
	user := &User{}
	u.DB.Where("email = ?", email).First(&user)
	return user
}

func (u *Users) GetById(id int) *User  {
	user := &User{}
	u.DB.Where("id = ?", id).First(&user)
	return user
}

func (u *Users) VerifyCredentials(email string, password string) (bool, *User)  {
	user := &User{}
	u.DB.Where("email = ?", email).First(&user)
	
	if user.Email != email {
		fmt.Println("Email does not exist")
		return false, user
	}
	fmt.Println(user.PasswordHash, user.Email)
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))

	return err == nil, user
}


func (u *Users) LoggedIn(user *User) {
	u.DB.Model(user).Update(map[string]interface{}{
		"last_logged_in_at": time.Now(),
		"last_user_agent": user.UserAgent,
		"last_ip": user.IP,
	})
}

func (u *Users) Update(user *User) error {
	if user.ID == 0 {
		// tmp := u.GetByEmail(user.Email)
	} else {
		u.DB.Save(&user)
	}
	return nil
}

func (u *Users) Create(user *User) error {
	if u.EmailExists(user.Email) {
		return errors.New("This email is already registered")
	}

	if len(user.Password) < 10 {
		return errors.New("The password is too short")
	}

	u.DB.Create(user)
	u.Emitter.Publish("user.created", user)
	return nil
}

type UserSession struct {
	ID  uuid.UUID `gorm:"type:varchar(36);primary_key;"`	
	UserID int
	IP string
	UserAgent string
	SessionID string `json:"-"`

	// Not stored in the database
	Browser string `gorm:"-"`
	OS string `gorm:"-"`
	IsMobile bool `gorm:"-"`
	IsCurrent bool `gorm:"-"`

}

func (u *UserSession) AfterFind() (err error) {
	ua := user_agent.New(u.UserAgent)
	name, version := ua.Browser()
	u.Browser =  name + " " + version
	u.OS = ua.OS()
	u.IsMobile = ua.Mobile()
	return
}

/*
// Clean up expired sessions
DELETE FROM user_sessions 
WHERE session_id in (select id from sessions where expires_at < NOW())
*/


// BeforeCreate will set a UUID rather than numeric ID.
func (j *UserSession) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", uuid.NewV4())
}
