package guac

import (
	"bytes"
	"log"
	"net"
	"strconv"
	"strings"
)

type Instruction struct {
	Inst    *[]byte
	Payload *[]byte
}

// func parseInstruction(s string) Instruction {
// 	index := strings.IndexRune(s, '.')
// 	instLength, _ := strconv.Atoi(s[:index])

// 	strInstr := s[index+1 : index+1+instLength]

// 	b := []byte(s)

// 	return Instruction{
// 		Inst:    &strInstr,
// 		Payload: &b,
// 	}
// }

func parseInstructionByte(b []byte) Instruction {
	index := bytes.IndexRune(b, '.')
	instLength, _ := strconv.Atoi(string(b[:index]))
	instruction := b[index+1 : index+1+int(instLength)]

	return Instruction{
		Inst:    &instruction,
		Payload: &b,
	}

}

func (instruction *Instruction) getInstruction() []byte {
	return *instruction.Inst
}

func (instruction *Instruction) getInstructionString() string {
	return string(*instruction.Inst)
}

func (instruction *Instruction) getPayload() []byte {
	return *instruction.Payload
}

func (instruction *Instruction) getPayloadParameters() []string {
	var params []string
	s := string(*instruction.Payload)

	index := strings.IndexRune(s, '.')
	instLen := 0
	for index > 0 {
		instLength, _ := strconv.Atoi(s[:index])

		strInstr := s[index+1 : index+1+instLength]
		// log.Println(strInstr)
		params = append(params, strInstr)

		instLen = strings.IndexRune(s[index:], ',')
		instLen = instLen + 1 + index
		index = strings.IndexRune(s[instLen:], '.')
		s = s[instLen:]
	}
	return params
}

/*func addToPayload([]byte payload, string s) []byte {

}*/

//MakeInstruction given strings as a parameter packs them in a byte slice to be further sent to TCP
func MakeInstruction(instructions ...string) []byte {
	var b []byte

	for i, instr := range instructions {
		if i != 0 {
			b = append(b, ',')

		}
		b = append(b, []byte(strconv.Itoa(len(instr)))...)
		b = append(b, '.')
		b = append(b, []byte(instr)...)
	}
	b = append(b, ';')
	return b
}

func MakeParameterString(param string) []byte {
	s := strconv.Itoa(len(param)) + "." + param + ","
	return []byte(s)
}

func MakeParameterInt(param int) []byte {
	s := strconv.Itoa(len(strconv.Itoa(param))) + "." + strconv.Itoa(param) + ","
	return []byte(s)
}

func SendInstruction(instruction string, conn net.Conn) {
	s := []byte(instruction)
	conn.Write(s)

	log.Printf("sent %v", string(s))
}

//given handshake byte slice from guacd, return response based on connection params
func MakeInstructionHandshake(params []string, connectionParams ConnectionParams) []byte {
	var b []byte

	b = append(b, []byte("5.audio;5.video;5.image;")...)

	for _, each := range params {
		switch each {
		case "args":
			b = append(b, []byte("7.connect,")...)

		case "hostname":
			b = append(b, MakeParameterString(connectionParams.RdpHostname)...)

		case "port":
			b = append(b, MakeParameterString(connectionParams.RdpPort)...)

		case "username":
			b = append(b, MakeParameterString(connectionParams.RdpUsername)...)

		case "password":
			b = append(b, MakeParameterString(connectionParams.RdpPassword)...)

		case "width":
			b = append(b, MakeParameterInt(connectionParams.DisplayWidth)...)

		case "height":
			b = append(b, MakeParameterInt(connectionParams.DisplayHeight)...)

		case "dpi":
			b = append(b, MakeParameterInt(connectionParams.DisplayDensity)...)

		case "security":
			b = append(b, MakeParameterString("")...)

		case "ignore-cert":
			b = append(b, MakeParameterString("true")...)

		default:
			b = append(b, []byte("0.,")...)
		}

	}
	b = b[:len(b)-1]
	b = append(b, []byte(";")...)
	// log.Println(string(b))
	return b
}

//switch case for connection params and received parameters required from guacd
func ParseHandshake(s string) []string {
	var params []string

	index := strings.IndexRune(s, '.')
	instLen := 0
	for index > 0 {
		instLength, _ := strconv.Atoi(s[:index])

		strInstr := s[index+1 : index+1+instLength]
		// log.Println(strInstr)
		params = append(params, strInstr)

		instLen = strings.IndexRune(s[index:], ',')
		instLen = instLen + 1 + index
		index = strings.IndexRune(s[instLen:], '.')
		s = s[instLen:]
	}
	return params
}
