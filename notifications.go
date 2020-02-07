package gitmon


type NotificationForGitFound struct {
	Scan Scan
	Config NotificationConfig
}

// https://api.slack.com/messaging/webhooks#posting_with_webhooks
type NotificationConfig struct {
	ID int 
	UserID int `json:"-" gorm:"unique_index:idx_user_org"`
	User User `json:"-" gorm:"association_autoupdate:false;association_autocreate:false"`
	OrganizationID int `json:"-" gorm:"unique_index:idx_user_org"`

	IsEmailEnabled bool
	IsSlackEnabled bool
	IsPagerDutyEnabled bool
	Email string `json:"email" validate:"email"`
	Slack string `json:"slack" validate:"url"`
	PagerDuty string `json:"pagerduty"`
}
