package gitmon 

import (
	"time"
	"net"
	"net/url"
	"net/http"
	"net/http/httptrace"
	"crypto/tls"
	"net/http/cookiejar"
)
// https://github.com/tcnksm/go-httpstat

// @todo requestor that emits events
func tracerWithStats() (*RequestStats, *httptrace.ClientTrace) {
	stats := &RequestStats{Started: time.Now()}
	trace := &httptrace.ClientTrace{
		DNSStart: func(dnsInfo httptrace.DNSStartInfo) {
			stats.StartedDNS = time.Now()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			stats.EndedDNS = time.Now()
		},
		ConnectStart: func(network, addr string) {
			stats.StartedConnect = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			stats.IP = addr
			stats.EndedConnect = time.Now()
		},
		TLSHandshakeStart: func() {
			stats.StartedTLS = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			stats.EndedTLS = time.Now()
		},
		GotFirstResponseByte: func() {
			stats.FirstByte = time.Now()
		},
	}
	return stats, trace
}


type UrlStatus struct {
	URL        *url.URL
	StatusCode int
}

type tweakedTransport struct {
	*http.Transport
	StatusCodes []UrlStatus
}

func (t *tweakedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.Transport.RoundTrip(request)
	statusCode := 0
	if err == nil {
		statusCode = response.StatusCode
	}
	t.StatusCodes = append(t.StatusCodes, UrlStatus{request.URL, statusCode})
	return response, err
}

func getTransport() *tweakedTransport {
	// @todo option to proxy with tor
	// https://www.devdungeon.com/content/making-tor-http-requests-go
	statusCodes := make([]UrlStatus, 0)

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   8 * time.Second,
			KeepAlive: 8 * time.Second,
			DualStack: true,
		}).DialContext,
		DisableKeepAlives: true,
		MaxIdleConns:          10,
		IdleConnTimeout:       10 * time.Second,
		ExpectContinueTimeout: 2 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
	}

	// @todo revisit ... catch cert errors?
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	return &tweakedTransport{transport, statusCodes}
}

type RequestStats struct {
	IP string
	Started  time.Time
	FirstByte time.Time
	StartedDNS time.Time
	EndedDNS time.Time
	StartedTLS time.Time
	EndedTLS time.Time
	StartedConnect  time.Time
	EndedConnect time.Time
	Ended time.Time
}

func (s RequestStats) Calculate()  Timings {
	data := make(Timings)
	data["total_time"] = s.Ended.Sub(s.Started)
	data["dns_time"] = s.EndedDNS.Sub(s.StartedDNS)
	data["tls_time"] = s.EndedTLS.Sub(s.StartedTLS)
	data["connect_time"] = s.EndedConnect.Sub(s.StartedConnect)
	return data
}

func addDefaultHeaders(request *http.Request) {
	headers := map[string]string{
		"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3",
		// "Accept-Encoding": "gzip, deflate",
		"Accept-Language": "en-US,en;q=0.9",
		"Cache-Control": "max-age=0",
		"Connection": "keep-alive",
		"Sec-Fetch-Mode": "navigate",
		"Sec-Fetch-Site": "same-origin",
		"Sec-Fetch-User": "?1",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.87 Safari/537.36",
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
}

func DoRequest(request *http.Request) (*http.Response, *RequestStats) {
	addDefaultHeaders(request)
	redirects := 0
	jar, _ := cookiejar.New(nil)
	client := http.Client{
		Timeout:   time.Second * 10,
		Jar:       jar,
		Transport: getTransport(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			addDefaultHeaders(req)
			redirects += 1
			if redirects <= 3 {
				return nil
			}
			return http.ErrUseLastResponse
		},
	}
	// https://blog.golang.org/http-tracing
	stats, trace := tracerWithStats()
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), trace))
	response, err := client.Do(request)

	if err != nil {
		panic(err) // @todo
	}
	stats.Ended = time.Now()
	return response, stats
}
