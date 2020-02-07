package gitmon

import  (
	"time"
	"net/url"
	"net/http"
)

func ScannerIsAlive() Scanner {
	return Scanner {
		ID: 1,
		Slug: "is_alive",
		Version: "1.0",
		Label: "Scan your site for exposed git repos",
		Tags: []string{"exposure"},
		Handle: HandleIsAlive,
		Interval: time.Duration(5) * time.Minute,
	}
}

func HandleIsAlive(scan *Scan, engine *ScanEngine) *Scan {
	target, _ := url.Parse(scan.Site.Url)
	request, err := http.NewRequest("GET", target.String(), nil)
    request.Close = true

	if err != nil {
		scan.Error = err.Error()
		return scan
	}

	response, stats := engine.HandleRequest(request)
	defer response.Body.Close() 
	scan.IP = stats.IP
	scan.Timings = stats.Calculate()
	scan.Status = 1
	scan.Result = 1

	if err != nil {
		scan.Status = 9
		scan.Error = err.Error()
		return scan
	}
	scan.StatusCode = response.StatusCode
	scan.Server = ExtractServer(response)
	// scan.Title = ExtractResponseTitle(response)
	
	id := engine.RecordTransaction(*scan, request, response)
	scan.RefID = &id
	response.Body.Close()
	return scan
}

