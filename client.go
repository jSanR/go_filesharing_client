package main

//Archivo con la función main del cliente

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

//Constantes
const NUMBER_OF_CHANNELS = 8   //Cantidad de canales disponibles para que un cliente se suscriba
const BUFFER_SIZE = 1024       //Tamaño de un buffer temporal utilizado para leer archivos iterativamente
const SERVER_PORT = "7101"     //Puerto en el que opera el servidor
const FILENAME_MAX_LENGTH = 40 //Tamaño máximo del nombre de un archivo que se recibe

func main() {
	//Verificar argumentos
	if len(os.Args) < 5 || len(os.Args) > 6 {
		fmt.Print("File sharing client: Send and receive files using channels through a TCP server\n\n")
		fmt.Println("Usage:")
		fmt.Println("Receive mode:\t client receive -channel CHANNEL -path DOWNLOAD_PATH")
		fmt.Println("Send mode:\t client send FILE -channel CHANNEL")
		fmt.Println("\nExamples:")
		fmt.Println("client receive -channel 3 -path D:\\Downloads\\ //Receive files sent by other clients to channel 3, saving them to selected download path")
		fmt.Println("client send test.txt -channel 4 //Send file test.txt to clients currently subscribed to channel 4")
		os.Exit(0)
	}
	//Verificar flags
	validateFlags()

	var mode string = os.Args[1]
	var channel int8
	var filepath string
	//Determinar el modo seleccionado por el cliente
	switch mode {
	case "receive":
		//Leer canal y, opcionalmente, path de descarga de archivos
		channel = parseChannel(os.Args[3])
		var downloadPath string = parseDownloadPath(os.Args[5])

		subscribeToChannel(channel, downloadPath)
	case "send":
		//Leer canal y path del archivo a enviar
		channel = parseChannel(os.Args[4])
		filepath = os.Args[2]

		sendFileThroughChannel(channel, filepath)
	default:
		fmt.Println("ERROR: Invalid command \"" + mode + "\"")
		os.Exit(1)
	}
}

func validateFlags() {
	switch len(os.Args) {
	case 5:
		//Only validate channel flag
		if os.Args[3] != "-channel" {
			fmt.Println("ERROR: Incorrect flag (expected \"-channel\", got \"" + os.Args[3] + "\")")
			os.Exit(1)
		}
	case 6:
		//Validate both channel and path flags
		if os.Args[2] != "-channel" {
			fmt.Println("ERROR: Incorrect flag (expected \"-channel\", got \"" + os.Args[2] + "\")")
			os.Exit(1)
		}
		if os.Args[4] != "-path" {
			fmt.Println("ERROR: Incorrect flag (expected \"-path\", got \"" + os.Args[4] + "\")")
			os.Exit(1)
		}
	}
}

//Función para parsear el canal a partir de un string y verificar su validez
func parseChannel(channelStr string) int8 {
	//Conversión del canal a un entero
	channel, parseError := strconv.Atoi(channelStr)
	if parseError != nil {
		fmt.Println("ERROR: Channel is not a valid number: " + parseError.Error())
		os.Exit(1)
	}
	//Se verifica un canal válido (el valor máximo se verifica al conectarse con el servidor)
	if channel < 1 || channel > NUMBER_OF_CHANNELS {
		fmt.Println("ERROR: Channel is outside valid range (1-" + strconv.Itoa(NUMBER_OF_CHANNELS) + ")")
		os.Exit(1)
	}
	return int8(channel)
}

func parseDownloadPath(path string) string {
	//Revisar que el path recibido contenga un separador de directorio al final
	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}
	//Validar que el path sea válido (básicamente, que exista el directorio
	_, err := os.Stat(path[:len(path)-1])
	if os.IsNotExist(err) {
		fmt.Println("ERROR: Invalid path")
		os.Exit(1)
	}
	return path
}
