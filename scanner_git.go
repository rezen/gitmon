package gitmon

import  (
	"time"
	"bytes"
	"strings"
	"fmt"
	"io/ioutil"
	"net/url"
	"path"
	"net/http"
)

func GitScanner() Scanner {
	return Scanner {
		ID: 2,
		Slug: "git_scan",
		Version: "1.0",
		Label: "Scan your site for exposed git repos",
		Tags: []string{"exposure"},
		Handle: HandleGitScan,
		Interval: time.Duration(10) * time.Minute,
	}
}

func HandleGitScan(scan *Scan, engine *ScanEngine) *Scan {
	target, _ := url.Parse(scan.Site.Url)
	target.Path = path.Join(target.Path, ".git/HEAD")
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

	if err != nil {
		scan.Status = 9
		scan.Error = err.Error()
		return scan
	}
	scan.StatusCode = response.StatusCode
	scan.Server = ExtractServer(response)
	/*
	timer := time.AfterFunc(5*time.Second, func() {
    	resp.Body.Close()
	})
	*/

	if scan.StatusCode == 200 {
		// @todo don't read it all!
		// 	lr := io.LimitReader(r, 4)
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println(err)
		}
		response.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		scan.ResponseSize = len(body)
		if scan.ResponseSize < 400 && bytes.Contains(body, []byte("ref:")) {
			scan.Result = 3
			scan.Body = strings.TrimSpace(string(body))
		}
	}
	if scan.Result != 3 {
		// scan.Title = ExtractResponseTitle(response)
	}
	response.Body.Close()
	return scan
}

