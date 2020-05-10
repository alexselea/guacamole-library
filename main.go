package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"guacamole-library/guac"

	"guacamole-library/redis"

	"github.com/spf13/viper"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var userPods map[string]*guac.UserConfig

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
}

var endpointIP string

var active int = 1

func upgradeWS(w http.ResponseWriter, r *http.Request) {

}

var endpointsArray []string

func getHostnames() {

	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	var endpoints []string
	for _, container := range containers {
		if strings.Contains(container.Names[0], "endpoint") {
			h := container.Names[0]

			endpoints = append(endpoints, h[1:])
		}
	}
	endpointsArray = endpoints
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	width := r.URL.Query()["width"]
	widthParam, _ := strconv.Atoi(width[0])
	height := r.URL.Query()["height"]
	heightParam, _ := strconv.Atoi(height[0])

	username := r.URL.Query()["username"]
	fmt.Println(username)

	//TODO: check if username is mapped with an existing connection
	hostname, connectionID := redis.GetConn(username[0])
	fmt.Println("hostname, connectionID: ", hostname, connectionID)

	if hostname == "" {
		getHostnames()
		hostname = endpointsArray[active]
		active++
	}
	fmt.Println("hostname, connectionID: ", hostname, connectionID)

	respHeader := make(http.Header)
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, respHeader)
	if err != nil {
		log.Println(err)
		return
	}

	guacdAddr, err := net.ResolveTCPAddr("", "guacd:4822")
	if err != nil {
		log.Println(err)
	}
	fmt.Println(viper.GetString("protocol"))

	connectionParams := &guac.ConnectionParams{
		Protocol:       viper.GetString("protocol"),
		GuacdAddr:      guacdAddr,
		RdpHostname:    hostname,
		RdpPort:        viper.GetString("rdp_port"),
		DisplayWidth:   widthParam,
		DisplayHeight:  heightParam,
		DisplayDensity: viper.GetInt("display_density"),
		RdpUsername:    viper.GetString("rdp_username"),
		RdpPassword:    viper.GetString("rdp_password"),
		ConnectionID:   connectionID,
		Username:       username[0],
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

func initViper() {
	viper.SetDefault("protocol", "rdp")
	viper.SetDefault("guacd_port", 4822)
	viper.SetDefault("rdp_port", "3389")
	viper.SetDefault("display_density", 96)
	viper.SetDefault("rdp_username", "alpine")
	viper.SetDefault("rdp_password", "alpine")
	viper.SetDefault("guacd_address", "127.0.0.1")
	viper.SetDefault("rdp_hostname", "endpoint-")
	viper.SetDefault("redis_hostname", "redis")
	viper.SetDefault("guacd_hostname", "guacd")
}

func initDockerClient() {
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}
}

// Create Container spawns a new instance of the endpoint
func spawnEndpoint() string {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		fmt.Println(err)
	}

	imageNameEndpoint := "danielguerra/alpine-xfce4-xrdp"

	var str strings.Builder
	str.WriteString("licenta_endpoint_")
	str.WriteString(strconv.Itoa(active))
	active++

	containerName := str.String()
	fmt.Println("containerName: ", containerName)
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image:    imageNameEndpoint,
			Hostname: containerName,
		},
		nil, nil, containerName)

	if err != nil {
		fmt.Println(err)
	}
	containerID := resp.ID
	fmt.Println("containerID: ", containerID)

	err = cli.NetworkConnect(ctx, "licenta_default", containerID, nil)
	if err != nil {
		fmt.Println(err)
	}

	return containerName
}

func main() {
	initViper()
	initDockerClient()

	redis.InitRedisClient()

	http.HandleFunc("/", staticHandler)

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
