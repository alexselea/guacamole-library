package guac

import (
	"bytes"
	"io"
	"log"
	"net"
	"strconv"

	red "guacamole-library/redis"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

var (
	Active           bool
	ConnectionGlobal *Connection
	Disconnected     bool
	Connections      map[string][]*Connection
)

type UserConfig struct {
	Email  string
	IP     string
	Uid    uuid.UUID
	ReqID  string
	Config string
	Active int
}

type ConnectionParams struct {
	GuacdAddr         *net.TCPAddr
	Protocol          string
	RdpHostname       string
	RdpPort           string
	DisplayWidth      int
	DisplayHeight     int
	DisplayDensity    int
	RdpUsername       string
	RdpPassword       string
	Security          string
	IgnoreCert        bool
	ConnectionID      string
	ActiveConnections int
	Username          string
}

type Connection struct {
	ConnParams      ConnectionParams
	ToFrontend      chan []byte
	ToGuacd         chan []byte
	WsConnection    *websocket.Conn
	GuacdConnection *net.TCPConn
	GuacMessages    chan string
	BeforeFilters   []func(string) bool
	AfterFilters    []func(string) bool
	Active          bool
}

func NewConnection(params ConnectionParams, ws *websocket.Conn) (*Connection, error) {
	if Connections == nil {
		Connections = make(map[string][]*Connection)
	}

	guacdConn, err := net.DialTCP("tcp", nil, params.GuacdAddr)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}

	guacConnection := &Connection{
		ConnParams:      params,
		ToFrontend:      make(chan []byte),
		ToGuacd:         make(chan []byte),
		WsConnection:    ws,
		GuacdConnection: guacdConn,
		GuacMessages:    make(chan string),
		Active:          true,
	}

	return guacConnection, nil
}

func (connection *Connection) Serve() {
	//start guacd handshake

	log.Printf("%+v\n", *connection)
	log.Printf("%+v\n", *&connection.ConnParams)
	if *&connection.ConnParams.ConnectionID == "" {
		selectInstruction := MakeInstruction("select", connection.ConnParams.Protocol)
		connection.GuacdConnection.Write(selectInstruction)

	} else {
		selectInstruction := MakeInstruction("select", connection.ConnParams.ConnectionID)
		connection.GuacdConnection.Write(selectInstruction)
	}

	guacdHandshake := make([]byte, 2000)
	connection.GuacdConnection.Read(guacdHandshake)

	params := ParseHandshake(string(guacdHandshake))

	clientHandshake := MakeInstruction("size", strconv.Itoa(connection.ConnParams.DisplayWidth),
		strconv.Itoa(connection.ConnParams.DisplayHeight), strconv.Itoa(connection.ConnParams.DisplayDensity))

	SendInstruction(string(clientHandshake), connection.GuacdConnection)

	connection.GuacdConnection.Write(MakeInstructionHandshake(params, connection.ConnParams))

	go connection.readFromGuacd()
	go connection.readFromWS()
	// go connection.sendToGuacd()
	// go connection.sendToWS()
	// go connection.fromGuacd()
	// go connection.fromWS()
}

func (connection *Connection) Close() {
	if connection.Active {
		b := MakeInstruction("disconnect")
		connection.ToFrontend <- b

		connection.GuacdConnection.Close()
	}
}

func (connection *Connection) readFromGuacd() {
	var buf []byte

	for connection.Active {
		s := make([]byte, 8192)
		connection.GuacdConnection.Read(s)

		s = bytes.Trim(s, "\x00")

		buf = append(buf, s...)

		index := bytes.LastIndex(buf, []byte(";"))
		// index := strings.LastIndex(string(buf), ";")

		if index != -1 {
			connection.WsConnection.WriteMessage(websocket.TextMessage, buf[:index+1])
			// connection.ToFrontend <- buf[:index+1]
			inst := parseInstructionByte((buf[:index+1]))
			// log.Println("Guacd to channel: ", inst.getInstruction(), string(inst.getPayload()))

			if bytes.Contains(buf[:index+1], []byte("disconnect")) {
				Disconnected = true
			}

			if bytes.Contains(buf[:index+1], []byte("ready")) {
				parameters := inst.getPayloadParameters()
				connection.ConnParams.ConnectionID = *&parameters[1]
				log.Println(connection.ConnParams.ConnectionID)

				username := connection.ConnParams.Username
				hostname := connection.ConnParams.RdpHostname

				red.PutNewConnectionOf(username, hostname, connection.ConnParams.ConnectionID)

			}
			if bytes.Contains(buf[:index+1], []byte("error")) {
				connection.GuacMessages <- string(*inst.Payload)
			}
		}
		//add remaining chunk of instructions to buffer
		buf = buf[index+1:]
	}
}

func (connection *Connection) readFromGuacd2() {
	for connection.Active {
		_, err := io.Copy(connection.GuacdConnection, connection.WsConnection.UnderlyingConn())
		if err != nil {
			log.Println(err)
			break
		}
	}
}

func (connection *Connection) readFromWS2() {
	for connection.Active {
		_, err := io.Copy(connection.WsConnection.UnderlyingConn(), connection.GuacdConnection)
		if err != nil {
			log.Println(err)
			break
		}
		_, err = io.Copy(connection.GuacdConnection, connection.WsConnection.UnderlyingConn())
		if err != nil {
			log.Println(err)
			break
		}

	}
}

//this function uses io.Copy to send streams from Guacd to WS
// func (connection *Connection) fromGuacd() {
// 	for connection.Active {
// 		writer, err := connection.WsConnection.NextWriter(websocket.TextMessage)

// 		written, err := io.Copy(writer, connection.GuacdConnection)
// 		if err != nil {
// 			log.Println(err)
// 		}
// 		log.Println("Written to WS: ", written)

// 	}

// }

//this function uses io.Copy to send streams from WS to Guacd
// func (connection *Connection) fromWS() {

// 	for connection.Active {
// 		//get io.Reader from WS connection
// 		_, reader, err := connection.WsConnection.NextReader()
// 		if err != nil {
// 			log.Println("read: ", err)
// 			break
// 		}

// 		delimWriter = net.

// 		written, err := io.Copy(connection.GuacdConnection, reader)
// 		if err != nil {
// 			panic(err)
// 		}
// 		log.Println("Written to guacd: ", written)
// 	}
// }

func (connection *Connection) sendToWS() {
	for connection.Active {

		select {
		case s := <-connection.ToFrontend:
			// log.Println("Channel to WS: ", string(s))
			// connection.WsMutex.Lock()
			connection.WsConnection.WriteMessage(websocket.TextMessage, s)
			// connection.WsMutex.Unlock()
		default:
			// log.Println("Couldn't read any message from toFrontEnd")
		}
	}
}

func (connection *Connection) readFromWS() {
	for connection.Active {
		s := make([]byte, 1024)

		_, s, err := connection.WsConnection.ReadMessage()
		if err != nil {

			// log.Println("Error readFromWS: ", err)
			connection.GuacMessages <- "Error readFromWS"
			return
		}
		// io.Copy(connection.GuacdConnection, connection.WsConnection.UnderlyingConn())
		connection.GuacdConnection.Write(s)

	}
}

func (connection *Connection) sendToGuacd() {
	for connection.Active {
		select {
		case s := <-connection.ToGuacd:
			// log.Println("Channel to Guacd: ", string(s))

			// connection.GuacdMutex.Lock()
			connection.GuacdConnection.Write(s)
			// connection.GuacdMutex.Unlock()

		default:
			// log.Println("Couldn't read anything from toGuacd")
		}
	}
}

func (connection *Connection) AddBeforeFilter(f func(param string) bool) {
	connection.BeforeFilters = append(connection.BeforeFilters, f)
}

func (connection *Connection) addAfterFilter(f func(param string) bool) {
	connection.AfterFilters = append(connection.AfterFilters, f)
}
