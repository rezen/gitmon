package gitmon

import (
	"time"
	"github.com/jinzhu/gorm"
	// "github.com/asaskevich/EventBus"
)

type HttpResponse struct {
	ID int	
	SiteID *int
	URI string `gorm:"type:text"`
	Hash string `gorm:"type:varchar(32)"`
	Headers string `gorm:"type:text"`
	// Body string
	UpdatedAt time.Time
}

type Scan struct {
	ID int	
	ScannerID int `json:"scanner_id"`
	SiteID int `json:"-"`
	Site   Site `json:"site" gorm:"association_autoupdate:false"`

	Status int
	UserID int `json:"-"`
	User User `json:"-" gorm:"association_autoupdate:false"`

	StatusCode int `json:"status_code"`
	ResponseSize int `json:"response_size"`
	Title string `json:"title"`
	Server string `json:"server"`
	TimedOut bool `json:"timed_out"`
	Body string `json:"body"`
	Result int `json:"result"` // 0=didnt_finish, 1=info, 3=vuln
	IP string `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	Error string `json:"error"`
	ResponseHash string `json:"-"`
	Timings Timings `json:"timings" gorm:"type:text"`
	RefID *int
}


type Scans struct {
	DB *gorm.DB
	Emitter *BetterBus
}

func (s *Scans) ByUser(user User) (scans []Scan) {
	s.DB.Order("created_at desc").Where("user_id = ?", user.ID).Preload("Site").Find(&scans)
	return scans
}


func (s *Scans) ForSite(site Site) (scans []Scan) {
	s.DB.Order("updated_at desc").Where("user_id = ? AND site_id = ?", site.UserID, site.ID).Preload("Site").Find(&scans)
	return scans
}


func (s *Scans) Where(where map[string]interface{}) (scans []Scan) {
	s.DB.Order("created_at desc").Limit(200).Where(where).Preload("Site").Find(&scans)
	return scans
}


/*
select site_id, scanner_id, MAX(created_at) 
from scans  
group by site_id, scanner_id 
having MAX(created_at) > "2019-10-29 11:20:41.108204-07:00";

8|1|2019-10-29 11:27:31.512658-07:00
8|2|2019-10-29 11:27:31.505898-07:00

9|1|2019-10-29 11:27:26.691947-07:00
9|2|2019-10-29 11:27:26.700219-07:00

11|1|2019-10-29 11:21:21.20592-07:00
11|2|2019-10-29 11:21:21.197472-07:00

14|1|2019-10-29 11:21:22.562089-07:00
14|2|2019-10-29 11:21:22.553675-07:00
16|1|2019-10-29 11:27:24.945566-07:00
16|2|2019-10-29 11:27:24.944475-07:00
17|1|2019-10-29 11:27:30.342896-07:00
17|2|2019-10-29 11:27:30.334105-07:00
18|1|2019-10-29 11:27:29.181135-07:00
18|2|2019-10-29 11:27:29.178056-07:00
21|1|2019-10-29 11:21:19.759634-07:00
21|2|2019-10-29 11:21:19.767798-07:00
22|1|2019-10-29 11:29:41.104654-07:00
22|2|2019-10-29 11:29:41.108204-07:00

func (s *Scans) Due(site Site) (scans []Scan) {
	s.DB.Where("created_at < ?", then).Find(&sites)
	return scans
}
*/

func (s *Scans) Start(site *Site) *Scan {
	// @todo ensure there is not already a running scan
	scan := &Scan {
		Status: 0,
		Site: *site,
		SiteID: site.ID,
		UserID: site.UserID,
		Result: 0,
		CreatedAt: time.Now(),
	}
	s.DB.Set("gorm:association_autoupdate", false).Save(&scan)
	return scan
}

func (s *Scans) Save(scan *Scan) {
	if scan.Status > 0 {
		scan.Site.LastScannedAt = &scan.CreatedAt
		s.DB.Model(&scan.Site).Update("last_scanned_at", scan.CreatedAt)
	}
	if scan.Result == 3 {
		s.Emitter.Publish("scan.failed", scan)
	}
	s.DB.Set("gorm:association_autoupdate", false).Save(&scan)
}
