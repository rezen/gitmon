package gitmon

// https://github.com/golang/go/issues/29162
import  (
	"os"
	"fmt"
	"errors"
	"time"
	"crypto/md5"
	"encoding/json"
	"encoding/hex"
	"io"
	"io/ioutil"
	"bytes"
	"strconv"
	"net/http"
	"github.com/jinzhu/gorm"

	// "github.com/asaskevich/EventBus"
)

// var ErrNoLocation = errors.New("http: no Location header in response")

type ScanEngine struct {
	Scanners map[string]Scanner
	ScannersIdMap map[int]string
	Emitter *BetterBus
	Scans *Scans
	Sites *Sites
	DB *gorm.DB
}

func (s ScanEngine) RecordTransaction(scan Scan, request *http.Request, response *http.Response) int {
	body, _ := ioutil.ReadAll(response.Body)
	response.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	hasher := md5.New()
	hasher.Write(body)
	hashed := hasher.Sum(nil)
	hash := hex.EncodeToString(hashed)
	// @todo sort 
	mapB, _ := json.Marshal(response.Header)
	fmt.Println(string(mapB))
	// fmt.Println(string(body))
	fmt.Println(response.Request.URL)
	
	res := &HttpResponse{
		Headers: string(mapB),
		Hash: hash,
		URI: response.Request.URL.RequestURI(),
		SiteID: &scan.SiteID,
	}

	s.DB.Create(res)
	prefix := hash[0:2]
	os.MkdirAll("_data/body/" + prefix, os.ModePerm)

	out, err := os.Create("_data/body/" + prefix + "/" + hash + ".html")
	if err != nil {
		// panic?
	}
	defer out.Close()
	io.Copy(out, response.Body)
	response.Body.Close()
	return res.ID
}


func (s ScanEngine) HandleRequest(request *http.Request) (*http.Response, *RequestStats)  {
	response, stats := DoRequest(request)
	body, _ := ioutil.ReadAll(response.Body)
	response.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	hasher := md5.New()
	hasher.Write(body)
	hashed := hasher.Sum(nil)
	hash := hex.EncodeToString(hashed)
	fmt.Println(hash)
	return response, stats
}

func (s ScanEngine) ScannerTag(id int) string {
	// @todo
	return s.ScannersIdMap[id]
}

func (s *ScanEngine) AddScanner(scanner Scanner) {
	s.ScannersIdMap[scanner.ID] = scanner.Slug
	s.Scanners[scanner.Slug] = scanner
}

func (s ScanEngine) ScannerById(id int) Scanner {
	slug := s.ScannerTag(id)
	return s.Scanners[slug]
}

func (s *ScanEngine) Scan(site *Site, scannerId int) error {
	scan := s.Scans.Start(site)
	scan.ScannerID = scannerId
	scanner := s.ScannerById(scannerId)

	return s.Execute(scan, scanner)
}

func (s *ScanEngine) ScansDue() []Scan {
	now := time.Now()
	due := []Scan{}

	for _, scanner := range s.Scanners {
		scans := []Scan{}
		then := now.Add(-1 * scanner.Interval)
		// .Having("MAX(created_at) > ?", then)
		s.DB.Table("scans").Select("MAX(site_id) as site_id, MAX(created_at) as created_at").Group("site_id").Where("scanner_id = ? AND created_at < ?", scanner.ID, then).Limit(400).Scan(&scans)
		for _, scan := range scans {
			due = append(due, Scan {
				ScannerID: scanner.ID,
				SiteID: scan.SiteID,
				Site: Site{ID: scan.SiteID},
			})
		}
	}
	return due
}

func (s *ScanEngine) Execute(scan *Scan, scanner Scanner) error {
	site := scan.Site

	if site.ID == 0 {
		return errors.New("Invalid site")
	}

	if len(site.Url) < 8 {
		site = *s.Sites.ById(strconv.Itoa(site.ID))
	}

	if len(site.Url) == 0 {
		return errors.New("Invalid site")
	}

	scan.Site = site
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in f", r)
        }
    }()

	// @todo wrap for catching errors
	scan = scanner.Handle(scan, s)
	s.Scans.Save(scan)
	return nil
}

func CreateScanEngineFromApp(app *App) *ScanEngine{
	return &ScanEngine{
		Scans: app.Scans,
		Sites: app.Sites,
		Emitter: app.Emitter,
		DB: app.DB,
		Scanners: map[string]Scanner{},
		ScannersIdMap: map[int]string{},
	}
}

type Scanner struct {
	ID int
	Slug string
	Version string
	Label string
	Emitter *BetterBus
	Tags []string
	Handle func(*Scan, *ScanEngine) *Scan
	Interval time.Duration
}

type ResponseScanner struct {
	Scanner
}