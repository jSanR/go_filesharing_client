package main

import (
	"fmt"
	"os"
	"strconv"
)

//Archivo con la función main del cliente

//Constantes
const BUFFER_SIZE = 1024       //Tamaño del buffer para enviar bytes al servidor
const SERVER_PORT = "7101"     //Puerto en el que opera el servidor
const FILENAME_MAX_LENGTH = 40 //Tamaño máximo del nombre de un archivo que se recibe
const RECEIVED_FILES_PATH = "D:\\Libraries\\Documentos\\testClient\\"

func main() {
	//Verificar argumentos
	if len(os.Args) != 4 && len(os.Args) != 5 {
		fmt.Println("Usage:")
		fmt.Println("Receive mode: client receive -channel CHANNEL (example: client receive -channel 1")
		fmt.Println("Send mode: cliente send FILE -channel CHANNEL (example: client send test.txt -channel 4")
		os.Exit(0)
	}

	var mode string = os.Args[1]
	var channel int8
	var filepath string
	switch mode {
	case "receive":
		channel = parseChannel(os.Args[3])
		subscribeToChannel(channel)
	case "send":
		channel = parseChannel(os.Args[4])
		filepath = os.Args[2]
		sendFileThroughChannel(channel, filepath)
	}
}

func parseChannel(channelStr string) int8 {
	channel, parseError := strconv.Atoi(channelStr)
	if parseError != nil {
		fmt.Println("ERROR: Channel is not a valid number: " + parseError.Error())
		os.Exit(1)
	}
	if channel < 1 {
		fmt.Println("ERROR: Channel is outside valid range (min: 1)")
		os.Exit(1)
	}
	return int8(channel)
}
