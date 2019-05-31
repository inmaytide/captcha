package eureka

import (
	"captcha/config"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
)

type Instance struct {
	ID             string
	BaseURL        string
	StatusURL      string
	RegisterAction HttpAction
	Registered     bool
}

var instances []Instance
var rawTpl []byte

func processTpl(env *config.Configuration, id string) string {
	tpl := string(rawTpl)
	tpl = strings.Replace(tpl, "${ip.address}", getLocalIP(), -1)
	tpl = strings.Replace(tpl, "${port}", fmt.Sprintf("%d", env.Application.Server.Port), -1)
	tpl = strings.Replace(tpl, "${instanceID}", id, -1)
	tpl = strings.Replace(tpl, "${application.name}", env.Application.Name, -1)
	return tpl
}

func makeInstance(env *config.Configuration, client config.EurekaClient) Instance {
	id := strings.Replace(uuid.NewV4().String(), "-", "", -1)
	baseURL := fmt.Sprintf("%s/apps/%s", client.ServiceUrl, env.Application.Name)
	tpl := processTpl(env, id)
	return Instance{
		ID:        id,
		BaseURL:   baseURL,
		StatusURL: fmt.Sprintf("%s/%s:%s:%s", baseURL, getLocalIP(), env.Application.Name, id),
		RegisterAction: HttpAction{
			URL:               baseURL,
			Method:            http.MethodPost,
			ContentType:       "application/json",
			Body:              tpl,
			HttpBasicUsername: client.Username,
			HttpBasicPassword: client.Password,
		},
		Registered: false,
	}
}

// Register ...
func Register(env *config.Configuration) {
	if len(env.Eureka.Clients) == 0 {
		log.Println("Couldn't found available eureka server...")
		return
	}

	instances = make([]Instance, len(env.Eureka.Clients))
	dir, err := os.Getwd()
	if err != nil {
		log.Println("Failed to get program root directory")
		log.Println(err)
		return
	}
	if rawTpl, err = ioutil.ReadFile(dir + "/conf/regtpl.json"); err != nil {
		log.Printf("Failed to read %s/conf/regtpl.json file \r\n", dir)
		log.Println(err)
		return
	}

	for i, n := range env.Eureka.Clients {
		instances[i] = makeInstance(env, n)
		go register(instances[i])
	}
}

func register(instance Instance) {
	for {
		if DoHttpRequest(instance.RegisterAction) {
			instance.Registered = true
			go heartbeat(instance)
			log.Printf("Registered: %s \r\n", instance.BaseURL)
			break
		} else {
			time.Sleep(time.Second * 5)
		}
	}
}

func heartbeat(instance Instance) {
	for {
		heartbeatAction := HttpAction{
			URL:               instance.StatusURL,
			Method:            "PUT",
			HttpBasicPassword: instance.RegisterAction.HttpBasicPassword,
			HttpBasicUsername: instance.RegisterAction.HttpBasicUsername,
		}
		if DoHttpRequest(heartbeatAction) {
			time.Sleep(time.Second * 30)
		} else {
			// log.Printf("Heartbeat failure: %s \r\n", instance.BaseURL)
			instance.Registered = false
			register(instance)
			break
		}

	}
}

// Deregister ...
func Deregister() {
	if len(instances) == 0 {
		return
	}
	log.Println("Trying to deregister application...")
	for _, n := range instances {
		if n.Registered {
			deregister(n)
		}
	}
	log.Println("Deregistered application, exiting. Check Eureka...")
}

func deregister(instance Instance) {
	deregisterAction := HttpAction{
		URL:    instance.StatusURL,
		Method: "DELETE",
	}
	DoHttpRequest(deregisterAction)
}

func getLocalIP() string {
	host, _ := os.Hostname()
	addrs, _ := net.LookupIP(host)

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ""
}
