package main
/**
   ServiceNow Integration
*/
import(
	"fmt"
	"github.com/magiconair/properties"
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"time"
	"io/ioutil"
	"os"
	"log"
	"encoding/json"
)

var MODE = ""
var SNurl = "https://XXX.service-now.com"
var user = ""
var password = ""
var EventClass = ""
var UMessageGroup = ""
var Source = ""
var CallerID = ""
var LOGLEVEL = ""
var proxies = ""
var http_port = ""
var PROXYBUONO = ""
func main() {

	propertyfile := "SNIntegrator.properties"
	prop := properties.MustLoadFile(propertyfile, properties.UTF8)
    
	MODE = prop.GetString("MODE", "")
	SNurl = prop.GetString("SNurl", "")
	user = prop.GetString("user", "")
	password = prop.GetString("password", "")
	EventClass = prop.GetString("EventClass", "")
	UMessageGroup = prop.GetString("UMessageGroup", "")
	Source = prop.GetString("Source", "")
	CallerID = prop.GetString("CallerID", "")
	LOGLEVEL = prop.GetString("LOGLEVEL", "")
	proxies = prop.GetString("proxies", "")
	http_port = prop.GetString("http_port", "")
	PROXYBUONO = proxyselect()
	
    switch MODE {
		case "CLI":
			executable(os.Args)
			
		case "HTTP":
			startServer()
		
		default:
			fmt.Printf("\n MODE incorrect or properties file  not found \n\n")
			os.Exit(2)
	}
	
}	
	
func executable(Args []string) {	
	start := time.Now()
	event_time := start.UTC().Format("2006-01-02 15:04:05")
	
   	argsWithoutProg := Args[1:]
	Prog := Args[0]

    if len(argsWithoutProg) < 1 {
		fmt.Printf("\n Usage: %s TASK 'TASK PARAMETER' \n\n TASKS:", Prog)
		fmt.Printf("\n    - incident \n    - alert_critical \n    - alert_info \n    - event_critical \n    - event_info \n\n")
		os.Exit(1)
    }
	
	action := Args[1]
	call_ok := false	
    
	hostname := ""
	if len(argsWithoutProg) > 2 {
		hostname = Args[2]
    }	
	


	switch action {

	case "incident":
		fmt.Println("incident")
        	if len(argsWithoutProg) != 5 {
			fmt.Printf("\n Usage of the Task %s: %s %s HOSTNAME GROUP SHORT_DESCRIPTION DESCRIPTION \n\n",action,Prog,action)
			os.Exit(1)
		}
		call_ok=openIncident(Args[3],Args[4],Args[5],event_time,hostname,PROXYBUONO) 

	case "alert_critical":
        	if len(argsWithoutProg) != 5 {
			fmt.Printf("\n Usage of the Task %s: %s %s HOSTNAME GROUP DESCRIPTION MESSAGE_KEY\n\n",action,Prog,action)
			os.Exit(1)
		}
		fmt.Println("alert critical")
		call_ok=openAlert(Args[3],Args[4],Args[5],"1",hostname,event_time,PROXYBUONO)

	case "alert_info":
        	if len(argsWithoutProg) != 5 {
			fmt.Printf("\n Usage of the Task %s: %s %s HOSTNAME GROUP DESCRIPTION MESSAGE_KEY\n\n",action,Prog,action)
			os.Exit(1)
		}
		fmt.Println("alert_info")
		call_ok=openAlert(Args[3],Args[4],Args[5],"5",hostname,event_time,PROXYBUONO)

	case "event_critical":
        	if len(argsWithoutProg) != 5 {
			fmt.Printf("\n Usage of the Task %s: %s %s HOSTNAME GROUP DESCRIPTION MESSAGE_KEY\n\n",action,Prog,action)
			os.Exit(1)
		}
		fmt.Println("event_critical")
		call_ok=openEvent(Args[3],Args[4],Args[5],"1","Ready",hostname,PROXYBUONO)

	case "event_info":
        	if len(argsWithoutProg) != 5 {
			fmt.Printf("\n Usage of the Task %s: %s %s HOSTNAME GROUP DESCRIPTION MESSAGE_KEY\n\n",action,Prog,action)
			os.Exit(1)
		}
		fmt.Println("event_info")
		call_ok=openEvent(Args[3],Args[4],Args[5],"5","Processed",hostname,PROXYBUONO)

	default:
		fmt.Printf("\n Task \"%s\" unknown \n\n",action)
		os.Exit(2)
	}

	if ! call_ok {
		fmt.Printf("\n Error executing Task \"%s\", if you need to configure a proxy to reach ServiceNow API create the file /etc/SN_proxy.conf with one or a list of comma separated Proxy \n\n",action)
                os.Exit(3)
	}
}

// ########
// INCIDENT
// ########
func openIncident(g string,sd string,d string,t string,ci string,proxyStr string) bool {
        ret := true
	type Payload struct {
		ShortDescription   string `json:"short_description"`
		Description        string `json:"description"`
		AssignmentGroup    string `json:"assignment_group"`
		CmdbCi             string `json:"cmdb_ci"`
		Urgency            string `json:"urgency"`
		UInizioDisservizio string `json:"u_inizio_disservizio"`
		Impact             string `json:"impact"`
		CallerID           string `json:"caller_id"`
	}
	data := Payload{
		ShortDescription: sd,
		Description: d,
		AssignmentGroup: g,
		CmdbCi: ci,
		Urgency: "1",
		UInizioDisservizio: t,
		CallerID: CallerID,
		Impact: "1"}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("ERROR",err)
		return false
	}
	body := bytes.NewReader(payloadBytes)
	
	req, err := http.NewRequest("POST", SNurl + "/api/now/table/incident", body)
	if err != nil {
		fmt.Printf("Errore",err)
		return false	
	}
	req.SetBasicAuth(user, password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	
	client := http.DefaultClient	


	if(proxyStr != ""){	
       		fmt.Println("Uso il proxy "+proxyStr)
    		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
        		fmt.Println("Errore nell'indirizzo del proxy")
			os.Exit(5)
		}
		transport := &http.Transport{ Proxy: http.ProxyURL(proxyURL) }
        client = &http.Client{Transport: transport}
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error",err)
		return false
	}
	defer resp.Body.Close()
	return ret
}

// ########
// ALERT
// ########
func openAlert(group string,desc string,mk string,severity string,node string,time string,proxyStr string) bool {
        ret := true
	
	type Payload struct {
		UMessageGroup string `json:"u_message_group"`
		Source        string `json:"source"`
		Severity      string `json:"severity"`
		CmdbCi        string `json:"cmdb_ci"`
		Description   string `json:"description"`
		MessageKey    string `json:"message_key"`
		MetricName    string `json:"metric_name"`
		Node          string `json:"node"`
		Time          string `json:"initial_event_time"`
	}
	
	data := Payload{
		UMessageGroup: UMessageGroup,
		Source: Source,
		Severity: severity,    
		CmdbCi: node,      
		Description: desc, 
		MessageKey: mk,  
		MetricName: group,
		Node: node,
        Time: time}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Errore",err)
		return false
	}
	body := bytes.NewReader(payloadBytes)
	
	req, err := http.NewRequest("POST", SNurl + "/api/now/table/em_alert", body)
	if err != nil {
		fmt.Printf("Error",err)
		return false
	}
	req.SetBasicAuth(user, password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	
	client := http.DefaultClient	

	if(proxyStr != ""){	
       		fmt.Println("Uso il proxy "+proxyStr)
    		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
        		fmt.Println("Errore nell'indirizzo del proxy")
			os.Exit(5)
		}
		transport := &http.Transport{ Proxy: http.ProxyURL(proxyURL) }
        	client = &http.Client{Transport: transport}
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Errore",err)
		return false
	}
	defer resp.Body.Close()

	return ret
}


// ########
// EVENT
// ########
func openEvent(group string,desc string,mk string,severity string,stato string,node string,proxyStr string) bool {
        ret := true
	
	type Payload struct {
		Description    string `json:"description"`
		Node           string `json:"node"`
		State          string `json:"state"`
		Classification string `json:"classification"`
		EventClass     string `json:"event_class"`
		Severity       string `json:"severity"`
		Source         string `json:"source"`
		MessageKey     string `json:"message_key"`
		MetricName     string `json:"metric_name"`
	}

	data := Payload{
		Description: desc,  
		Node: node,
		State: stato,
		Classification:	"IT",
		EventClass: EventClass,
		Severity: severity,
		Source: Source,
		MessageKey: mk, 
		MetricName: group } 

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Errore",err)
		return false
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", SNurl + "/api/now/table/em_event", body)
	if err != nil {
		fmt.Printf("Errore",err)
		return false
	}
	req.SetBasicAuth(user, password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient	


	if(proxyStr != ""){	
       		fmt.Println("Uso il proxy "+proxyStr)
    		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
        		fmt.Println("Errore nell'indirizzo del proxy")
			os.Exit(5)
		}
		transport := &http.Transport{ Proxy: http.ProxyURL(proxyURL) }
        	client = &http.Client{Transport: transport}
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Errore",err)
		return false
	}
	defer resp.Body.Close()
	return ret
}

// ##############
//
//

func proxyselect() string{
	proxylist := strings.Split(proxies, string(','))
	for i := 0 ; i < len(proxylist) ; i++ {
		proxy := strings.Trim(proxylist[i]," ")
		if LOGLEVEL == "DEBUG" { fmt.Printf("Provo il proxy %v \n",proxy)}
		if !strings.HasPrefix(proxy, "http") { proxy = "http://" + proxy }
		retbool,errore := checkProxy(proxy)
		if LOGLEVEL == "DEBUG" { fmt.Printf("Testato il proxy %v, ritorno %v - %v \n",proxy,retbool,errore)}
                if retbool {
			if LOGLEVEL == "DEBUG" { fmt.Printf("Trovato il proxy %v \n",proxy)}
			return proxy
		} 
	}	
	return ""

}

func checkProxy(proxy string) (success bool, errorMessage string) {	
	proxyUrl, err := url.Parse(proxy)
	timeout := time.Duration(3 * time.Second)
	httpClient := &http.Client { Transport: &http.Transport { Proxy: http.ProxyURL(proxyUrl) }, Timeout: timeout}
	response, err := httpClient.Get(SNurl)
	if err != nil { return false, err.Error() }

	body, err := ioutil.ReadAll(response.Body)
	if err != nil { return false, err.Error() }

	bodyString := strings.ToLower(strings.Trim(string(body), " \n\t\r"))

	if strings.Index(bodyString, "<body") < 0 && strings.Index(bodyString, "<head") < 0 {
		if strings.Index(bodyString, "<title>invalid request</title>") >= 0 {
			return false, "Tracker responsed 'Invalid request' - might be dead"
		} else {
			return false, "Received page is not HTML: " + bodyString
		}
	}

	return true, ""
}

func startServer() {
	fmt.Printf("Starting HTTP SERVER on port %s \n",http_port)
	http.HandleFunc("/", WebServerHandler)
    http.ListenAndServe(":"+http_port, nil)
}
type test_struct struct {
    Hostname string
}
func WebServerHandler(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Path[1:]
    log.Println("Action: " + action)
	body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        panic(err)
    }
    log.Println("Body: " + string(body))
    var t test_struct
    err = json.Unmarshal(body, &t)
    if err != nil {
        panic(err)
    }
    log.Println(t.Hostname)
	
}
