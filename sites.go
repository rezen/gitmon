package gitmon

import (
	"net"
	"net/url"
	"fmt"
	"time"
	"errors"
	"github.com/jinzhu/gorm"
	// "github.com/asaskevich/EventBus"

)

var privateIPBlocks []*net.IPNet

func init() {
    for _, cidr := range []string{
        "127.0.0.0/8",    // IPv4 loopback
        "10.0.0.0/8",     // RFC1918
        "172.16.0.0/12",  // RFC1918
        "192.168.0.0/16", // RFC1918
        "::1/128",        // IPv6 loopback
        "fe80::/10",      // IPv6 link-local
        "fc00::/7",       // IPv6 unique local addr
    } {
        _, block, err := net.ParseCIDR(cidr)
        if err != nil {
            panic(fmt.Errorf("parse error on %q: %v", cidr, err))
        }
        privateIPBlocks = append(privateIPBlocks, block)
    }
}

func isPrivateIP(ip net.IP) bool {
    for _, block := range privateIPBlocks {
        if block.Contains(ip) {
            return true
        }
    }
    return false
}

type SiteRequest struct {
	ID int `json:"-"`
	Url string `json:"url" validate:"required,url"`
	IsEnabled bool `json:"is_enabled"`
	Tags Tagset `json:"tags"`
}


func (s SiteRequest) ToSite() Site {
	site := Site{}
	site.ID = s.ID
	site.Url = s.Url
	site.Tags = s.Tags
	return site
}

type Site struct {
	ID int
	UserID int `json:"user_id" gorm:"unique_index:idx_user_sites"`
	User User `json:"-" gorm:"association_autoupdate:false"`
	Url string `json:"url" validate:"required,url" gorm:"unique_index:idx_user_sites"`
	IsEnabled bool `json:"is_enabled"`
	Tags Tagset `json:"tags" gorm:"type:varchar(100);"`
	Organization int `json:"-"`
	LastScannedAt *time.Time `json:"last_scanned_at" gorm:"default:null"`
	HasGitExposed bool `json:"has_git_exposed" param:"-"`
	Scans []Scan `json:"-"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}


func IsValidUrl(u string) bool {
	target, err := url.Parse(u)

	if target.Scheme != "http" && target.Scheme != "https" {
		return false
	}
	if err != nil {
		return false
	}

	if target.Host == "localhost" {
		return false
	}

	if target.Host == "169.254.169.254" {
		return false
	}

	ip := net.ParseIP(target.Host)

	if ip != nil {
		return !isPrivateIP(ip)
	}

	return true
}

type Sites struct {
	DB *gorm.DB
	Emitter *BetterBus
	// ScanInterval
}

func (s *Sites) ByUser(user User) (sites []Site) {
	s.DB.Where("user_id = ?", user.ID).Find(&sites)
	return sites
}

func (s *Sites) WithTagByUser(tag string, user User) (sites []Site) {
	s.DB.Where("user_id = ? AND tags LIKE ?", user.ID,  "%" + tag + "%").Find(&sites)
	// @todo filter results by tag
	return sites
}

func (s *Sites) SearchByUser(search string, user User) (sites []Site) {
	like :=  "%" + search + "%"
	s.DB.Where("user_id = ? AND (url LIKE ? or tags LIKE ?)", user.ID, like, like).Find(&sites)
	return sites
}


func (s *Sites) ById(id string) (*Site) {
	site := &Site{}
	s.DB.Where("id = ?", id).First(&site)
	return site
}

func (s *Sites) Create(site *Site) error {
	site.CreatedAt = time.Now()

	if !IsValidUrl(site.Url) {
		return errors.New("Invalid url")
	}
	count := 0
	s.DB.Where("user_id = ? AND url = ?", site.UserID, site.Url).First(&Site{}).Count(&count)

	if count > 0 {
		return errors.New("Already have this url")
	}

	s.DB.Create(site)
	s.Emitter.Publish("site.created", site)
	return nil
}


type PaginatedSite struct {
	Data []Scans
	Page int
	PerPage int
}
