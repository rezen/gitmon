package gitmon 

import (
	"net/http"
	"io"
	"bufio"
	"regexp"

)
func ExtractServer(response *http.Response) string {
	if len(response.Header["X-Powered-By"]) > 0 {
		return string(response.Header["X-Powered-By"][0])
	}

	if len(response.Header["Server"]) > 0 {
		return string(response.Header["Server"][0])
	}
	return ""
}

func ExtractResponseTitle(response *http.Response) string {
	// @todo have timeout
	/*
	scanner := bufio.NewScanner(response.Body)
	*/
	reader := bufio.NewReader(response.Body)

	maxLines := 200 // Should be very few in this camp
	regexForTitle := regexp.MustCompile("<title>(.*?)</title>")
	counter := 0
	for {
		line, _, err := reader.ReadLine()

		counter = counter + 1
		matches := regexForTitle.FindStringSubmatch(string(line))
		if len(matches) > 0 {
			return string(matches[1])
			break
		}

		if err == io.EOF {
			break
		}

		if counter == maxLines {
			break
		}
	}

	/*
	for scanner.Scan() {
		counter = counter + 1
		matches := regexForTitle.FindStringSubmatch(scanner.Text())
		if len(matches) > 0 {
			return matches[1]
			break
		}

		if counter == maxLines {
			break
		}
	}
	if err := scanner.Err(); err != nil {
        panic(err)
    }*/
	return ""
}
