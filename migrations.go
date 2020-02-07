package gitmon

import (
	"github.com/jinzhu/gorm"
)

func SetupDatabase(db *gorm.DB) {

	db.Exec(`CREATE TABLE IF NOT EXISTS state (
		name VARCHAR(140) NOT NULL,
		value TEXT NOT NULL,
		PRIMARY KEY (name)
	);`)
	db.AutoMigrate(&UserSession{})
	db.AutoMigrate(&User{})
	// db.AutoMigrate(&Job{})
	db.AutoMigrate(&PasswordReset{})
	db.AutoMigrate(&Site{})
	db.AutoMigrate(&Scan{})
	db.AutoMigrate(&NotificationConfig{})
	db.AutoMigrate(&HttpResponse{})
}
