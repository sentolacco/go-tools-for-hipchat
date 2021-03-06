package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"

	"bitbucket.org/atlassianlabs/hipchat-golang-base/util"

	"github.com/gorilla/mux"
	"github.com/tbruyelle/hipchat-go/hipchat"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/sentolacco/go-tools-for-hipchat/tools"
)

// RoomConfig holds information to send messages to a specific room
type RoomConfig struct {
	token *hipchat.OAuthAccessToken
	hc    *hipchat.Client
	name  string
}

// Context keep context of the running application
type Context struct {
	baseURL string
	static  string
	//rooms per room OAuth configuration and client
	rooms map[string]*RoomConfig
}

func (c *Context) healthcheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{"OK"})
}

func (c *Context) atlassianConnect(w http.ResponseWriter, r *http.Request) {
	lp := path.Join("./static", "atlassian-connect.json")
	vals := map[string]string{
		"LocalBaseUrl": c.baseURL,
	}
	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		log.Fatalf("%v", err)
	}
	tmpl.ExecuteTemplate(w, "config", vals)
}

func (c *Context) installableCallback(w http.ResponseWriter, r *http.Request) {
	authPayload, err := util.DecodePostJSON(r, true)
	if err != nil {
		log.Fatalf("Parsed auth data failed:%v\n", err)
	}

	credentials := hipchat.ClientCredentials{
		ClientID:     authPayload["oauthId"].(string),
		ClientSecret: authPayload["oauthSecret"].(string),
	}
	roomName := strconv.Itoa(int(authPayload["roomId"].(float64)))
	newClient := hipchat.NewClient("")
	tok, _, err := newClient.GenerateToken(credentials, []string{hipchat.ScopeSendNotification})
	if err != nil {
		log.Fatalf("Client.GetAccessToken returns an error %v", err)
	}
	rc := &RoomConfig{
		name: roomName,
		hc:   tok.CreateClient(),
	}
	c.rooms[roomName] = rc

	util.PrintDump(w, r, false)
	json.NewEncoder(w).Encode([]string{"OK"})
}

func (c *Context) tools(w http.ResponseWriter, r *http.Request) {
	payLoad, err := util.DecodePostJSON(r, true)
	if err != nil {
		log.Fatalf("Parsed auth data failed:%v\n", err)
	}
	roomID := strconv.Itoa(int((payLoad["item"].(map[string]interface{}))["room"].(map[string]interface{})["id"].(float64)))

	util.PrintDump(w, r, true)

	m1 := payLoad["item"].(map[string]interface{})
	m2 := m1["message"].(map[string]interface{})
	message := m2["message"].(string)
	fmt.Printf("Message: %v\n", message)

	result, err := tools.Tools(message)

	notifRq := &hipchat.NotificationRequest{
		Message:       "uninitialized",
		MessageFormat: "html",
		Color:         "red",
	}

	if err != nil {
		notifRq = &hipchat.NotificationRequest{
			Message:       "/code " + err.(*flags.Error).Message,
			MessageFormat: "text",
			Color:         "red",
		}
	} else {
		notifRq = &hipchat.NotificationRequest{
			Message:       "/code " + result,
			MessageFormat: "text",
			Color:         "green",
		}
	}

	log.Printf("Sending notification to %s\n", roomID)
	if _, ok := c.rooms[roomID]; ok {
		_, err = c.rooms[roomID].hc.Room.Notification(roomID, notifRq)
		if err != nil {
			log.Printf("Failed to notify HipChat channel:%v\n", err)
		}
	} else {
		log.Printf("Room is not registered correctly:%v\n", c.rooms)
	}
}

// routes all URL routes for app add-on
func (c *Context) routes() *mux.Router {
	r := mux.NewRouter()
	//healthcheck route required by Micros
	r.Path("/healthcheck").Methods("GET").HandlerFunc(c.healthcheck)
	//descriptor for Atlassian Connect
	r.Path("/").Methods("GET").HandlerFunc(c.atlassianConnect)
	r.Path("/atlassian-connect.json").Methods("GET").HandlerFunc(c.atlassianConnect)

	// HipChat specific API routes
	r.Path("/installable-callback").Methods("POST").HandlerFunc(c.installableCallback)
	r.Path("/tools").Methods("POST").HandlerFunc(c.tools)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(c.static)))
	return r
}

func main() {
	var (
		port    = flag.String("port", "8080", "web server port")
		static  = flag.String("static", "./static/", "static folder")
		baseURL = flag.String("baseurl", /*os.Getenv("BASE_URL")*/ "https://41efe6a3.ngrok.io", "local base url")
	)
	flag.Parse()

	c := &Context{
		baseURL: *baseURL,
		static:  *static,
		rooms:   make(map[string]*RoomConfig),
	}

	log.Printf("Base HipChat integration v0.10 - running on port:%v", *port)

	r := c.routes()
	http.Handle("/", r)
	http.ListenAndServe(":"+*port, nil)
}
