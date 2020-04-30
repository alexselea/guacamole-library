package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"./guac"

	"github.com/spf13/viper"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var userPods map[string]*guac.UserConfig

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
}

var endpointIP string

func serveWs(w http.ResponseWriter, r *http.Request) {
	width := r.URL.Query()["width"]
	widthParam, _ := strconv.Atoi(width[0])
	height := r.URL.Query()["height"]
	heightParam, _ := strconv.Atoi(height[0])

	username := r.URL.Query()["username"]
	fmt.Println(username)

	respHeader := make(http.Header)
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, respHeader)
	if err != nil {
		log.Println(err)
		return
	}

	guacdAddr, err := net.ResolveTCPAddr("", "guacd:4822")

	connectionParams := &guac.ConnectionParams{
		Protocol:       viper.GetString("protocol"),
		GuacdAddr:      guacdAddr,
		RdpHostname:    "endpoint",
		RdpPort:        viper.GetString("rdp_port"),
		DisplayWidth:   widthParam,
		DisplayHeight:  heightParam,
		DisplayDensity: viper.GetInt("display_density"),
		RdpUsername:    viper.GetString("rdp_username"),
		RdpPassword:    viper.GetString("rdp_password"),
	}

	guacConn, _ := guac.NewConnection(*connectionParams, conn)
	go guacConn.Serve()
}

func filter(s string) bool {
	if s == "mouse" {
		return true
	}
	return false
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	// ipQuery := r.URL.Query()["IP"]
	// endpointIP = ipQuery[0]
	// log.Println(endpointIP)

	fs := http.Dir(".")
	fileServer := http.FileServer(fs)
	cleanPath := path.Clean(r.URL.Path)

	_, err := fs.Open(cleanPath)
	if os.IsNotExist(err) && strings.HasPrefix(cleanPath, "/blog") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		log.Println("Serving 404 html")
		http.ServeFile(w, r, "www/blog/404.html")
		return
	}
	log.Println(r.URL)

	fileServer.ServeHTTP(w, r)
}

func main() {
	viper.SetDefault("protocol", "rdp")
	viper.SetDefault("guacd_port", 4822)
	viper.SetDefault("rdp_port", "3389")
	viper.SetDefault("display_density", 96)
	viper.SetDefault("rdp_username", "alpine")
	viper.SetDefault("rdp_password", "alpine")
	viper.SetDefault("guacd_address", "127.0.0.1")

	//initiate instance manager client

	http.HandleFunc("/", staticHandler)
	//call server http handler

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(w, r)
	})
	http.Handle("/metrics", promhttp.Handler())

	err := http.ListenAndServe("0.0.0.0:8888", nil)
	log.Println("Server started")
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	log.Println("Prometheus server started")
}
