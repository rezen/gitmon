package gitmon

import (
	"strings"
    "os"
    "os/user"
    "strconv"
    "log"
    "net"
)

func unique(items []string) []string {
    keys := make(map[string]bool)
    list := []string{} 
    for _, entry := range items {
		entry = strings.TrimSpace(entry)
        if _, value := keys[entry]; !value {
            keys[entry] = true
            list = append(list, entry)
        }
    }    
    return list
}

// Get preferred outbound ip of this machine
func GetOutboundIP() string {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
        return ""
    }
    defer conn.Close()
    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP.String()
}

type RunningDetails struct {
    Hostname string
    IP string
    PID string
    User string
}

func (r RunningDetails) String() string {
    return "hostname:" + r.Hostname + ";pid:" + r.PID + ";ip:" + r.IP + ";user:" + r.User
}

func GetRunningDetails() RunningDetails {
    name, _ := os.Hostname()
    user, _ := user.Current()

	ip := GetOutboundIP()
    pid := os.Getpid()
    return RunningDetails {
        Hostname: name,
        IP: ip,
        PID: strconv.Itoa(pid),
        User: user.Uid,
    }
}

/*
// https://rollbar.com/blog/error-monitoring-golang/
func recoverError() {   
    if r := recover(); r!= nil {
      fmt.Println(r)
      rollbar.Critical(r)
      rollbar.Wait()    
    }
}
*/