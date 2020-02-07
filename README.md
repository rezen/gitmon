### Todo
- https://github.com/steambap/captcha
- Middleware for blocking uncommon user agent
- Event emitter

### References
#### Helpful
- https://github.com/gothinkster/golang-gin-realworld-example-app/tree/master/users
- https://gowebexamples.com/sessions/
- https://www.alexsears.com/2019/10/fun-with-concurrency-in-golang/
- https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
- http://dahernan.github.io/2015/02/04/context-and-cancellation-of-goroutines/
- https://medium.com/@matryer/stopping-goroutines-golang-1bf28799c1cb
- https://github.com/eapache/go-resiliency



#### Tuning
- https://medium.com/@pawilon/tuning-your-linux-kernel-and-haproxy-instance-for-high-loads-1a2105ea553e
- http://blog.mact.me/2014/10/22/yosemite-upgrade-changes-open-file-limit
- https://www.imakewebsites.ca/posts/2018/03/06/node.js-too-many-open-files-and-ulimit/
- https://serverfault.com/questions/265155/soft-limit-vs-hard-limit
- https://blog.jayway.com/2015/04/13/600k-concurrent-websocket-connections-on-aws-using-node-js/
- https://mrotaru.wordpress.com/2013/10/10/scaling-to-12-million-concurrent-connections-how-migratorydata-did-it/
- https://blog.twitch.tv/en/2019/04/10/go-memory-ballast-how-i-learnt-to-stop-worrying-and-love-the-heap-26c2462549a2/

```bash
# Get file limits
ulimit -n

# Get process limits
ulimit -u
```

#### Debugging
```sh
go tool pprof -http=:8082 ./pprof/pprof.samples.cpu.001.pb.gz
```
- https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/
- https://matoski.com/article/golang-profiling-flamegraphs/
- https://blog.detectify.com/2019/09/05/how-we-tracked-down-a-memory-leak-in-one-of-our-go-microservices/
- https://matoski.com/article/golang-profiling-flamegraphs/
- https://go101.org/article/memory-leaking.html
- https://hackernoon.com/avoiding-memory-leak-in-golang-api-1843ef45fca8


#### Concurrency
- https://geeks.uniplaces.com/building-a-worker-pool-in-golang-1e6c0fdfd78c
- https://stackoverflow.com/questions/37774624/go-http-get-concurrency-and-connection-reset-by-peer
- https://medium.com/@greenraccoon23/multi-thread-for-loops-easily-and-safely-in-go-a2e915302f8b

#### Monitoring
```sh
curl http://localhost:8086/query --data-urlencode 'q=SHOW DATABASES'
```

#### Database
- https://blog.depado.eu/post/gorm-gotchas
- https://dba.stackexchange.com/questions/41872/configuring-mysql-my-cnf-for-myisam-and-innodb

#### Seed
```sh
[[ -f top-1m.csv.zip ]] || wget -q http://s3.amazonaws.com/alexa-static/top-1m.csv.zip
unzip top-1m.csv.zip
awk -F ',' '{print "https://"$2}' top-1m.csv | head -4000 > top-4000.txt
```

scan_schedules
    scanner_id, interval

site_scan_state
    site_id, scanner_id, last_ran, status, is_enabled

```go
/*
type HttpResponse struct {
    ID int	
    Initiator string
    SiteID *int
    IP string
    StatusCode int
    Timings Timings
    BodySize int
	URI string `gorm:"type:text"`
	Hash string `gorm:"type:varchar(32)"`
	Headers string `gorm:"type:text"`
	// Body string
    UpdatedAt time.Time
}

*/

type Factset struct {
    Facts []*HttpFacts
}

type HttpFacts struct {
    Response *http.Response
    BodyString string

    Hash string `json:"-"`
    Timings Timings
    ResponseSize int
    IP string
    Site Site
    Error error
    TimedOut bool
    Server string
    RefID string // uuid?
    // Query for css selectors
    Alerts []*Alert
    ScannedBy []*Scanner
}

type  Alert struct {
    Level string
    Label string
    Site *Site
    SiteID int
    Scanner *Scan
    ScannerID int
    RefID string,
    Meta map[string]string
}

type Alerter struct {
    DB *gorm.DB
}
func (a *Alerter) Raise(facts *HttpFacts, scanner *Scanner, level string, meta map[string]string) {
    // @todo there may be multiple facts that cause an alert
    alert := &Alert {
        Level: level,
        Label: scanner.Label,
        Scanner: scanner,
        ScannerID: scanner.ID
        Site: site,
        SiteID: site.ID,
        RefID: facts.RefID,
        Meta: meta,
    }
    a.DB.Create(alert)
    facts.Alerts =  append(facts.Alerts, alert)
}

type HttpConfig struct {
    Path string
    Headers map[string]string
    Body string
    Method string
}

func (h HttpConfig) ToRequest() *http.Request {
    request, _ := http.NewRequest(h.Method, h.Path, h.Body)
    for key, value := range h.Headers {
		request.Header.Set(key, value)
	}
    request.Close = true
    return request
}



type ResponseMatch struct {
    OnMethod string
    NotMethod string
    OnHost string
    NotHost string
    UrlContains string
    OnXML bool
    OnJSON bool
    OnForm bool
}

type Scan struct {
    Label string
    Wants []HttpConfig
    When []ResponseMatch
    Handler func(*HttpFacts, *Alerter) error
    Record bool // Always record site?
    Interval time.Duration
    // Precondition only against sites that have x in body
    // ArchivalRules ArchivalRules
}

type ScanExecutor struct {
    Scanners []Scan
    WhenScanners []Scan
    Alerter *Alerter
}


func (s ScanExecutor) Execute(scan Scan, site Site) {
    if len(scan.Wants) {
        return
    }

    for _, want := range scan.Wants {

        request := want.ToRequest()
        response, stats := s.Engine.HandleRequest(request)

        body, _ := ioutil.ReadAll(response.Body)
    	response.Body = ioutil.NopCloser(bytes.NewBuffer(body))
    	hasher := md5.New()
    	hasher.Write(body)
    	hashed := hasher.Sum(nil)
    
        fact := &HttpFacts {
            Site: site,
            Response: response,
            IP: stats.IP,
            RefID: "cats",
            Timings: stats.Calculate(),
            BodyString: string(body),
            Hash:  hex.EncodeToString(hashed)
        }

        scan.ApplyHandler(fact, scan)

        for _, sc := range s.WhenScanners {
            if s.ShouldHandle(fact, sc) {
                scan.ApplyHandler(fact, sc)
            }
        }
    }
}


func (s ScanExecutor) ShouldHandle(facts *HttpFacts, scan Scan)  bool {
    if len(scan.Wants) == 0 && len(scan.When) == 0 {
        return false
    }

    for _, when := range scan.When {
        if when.OnJSON && facts.IsJSON() {
            return true
        }
    }

    return false
}

func (s ScanExecutor) ApplyHandler(facts *HttpFacts, scan Scan) {
    err := scan.Handler(facts, s.Alerter)
    facts.ScannedBy = append(facts.ScannedBy, scan)
}


&Scan {
    Label: "Git exposed",
    Wants []HttpConfig{
        {Path: "git/HEAD"},
        {Path: "phpinfo.php"},
        {Path: "info.php"},
    },
    Handler: function(facts Factset, alerter Alerter) {
        fact := facts.ByPath("git/HEAD")
        if (strings.Contains(fact.BodyString, "ref:")) {
            alerter.Raise(fact, "HIGH", "Git exposed", map[string]string{

            })
        }
    }
}

&Scan {
    Label: "Good cookie usage",
    Wants []HttpConfig{
        {Path: "/"},
        {Path: "/robots.txt"},
    },
    Handler: function(facts Factset, alerter Alerter) {
        fact := facts.ByPath("git/HEAD")
        if (strings.Contains(fact.BodyString, "ref:")) {
            alerter.Raise(fact, "HIGH", "Git exposed", map[string]string{

            })
        }
    }
}


&WhenScanner {
    Label: "Checks json for stuff",
    When [
        &ResponseMatch{OnJSON: true},
    ],
    Handler: function(facts *HttpFacts, alerter Alerter) {

    }
}
```