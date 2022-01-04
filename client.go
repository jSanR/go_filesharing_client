package main

//Archivo con la función main del cliente

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

//Constantes
const BUFFER_SIZE = 1024       //Tamaño del buffer para enviar bytes al servidor
const SERVER_PORT = "7101"     //Puerto en el que opera el servidor
const FILENAME_MAX_LENGTH = 40 //Tamaño máximo del nombre de un archivo que se recibe
const DEFAULT_DOWNLOAD_PATH = "D:\\Libraries\\Documentos\\testClient\\"

func main() {
	//Verificar argumentos
	if len(os.Args) < 4 || len(os.Args) > 6 {
		fmt.Println("File sharing client: Send and receive files using channels through a TCP server\n")
		fmt.Println("Usage:")
		fmt.Println("Receive mode:\t client receive -channel CHANNEL [-path DOWNLOAD_PATH]")
		fmt.Println("Send mode:\t client send FILE -channel CHANNEL")
		fmt.Println("\nExamples:")
		fmt.Println("client receive -channel 1 //Receive files sent by other clients to channel 1 using the default download path")
		fmt.Println("client receive -channel 3 -path D:\\Downloads\\ //Receive files sent by other clients to channel 3 using a custom download path")
		fmt.Println("client send test.txt -channel 4 //Send file test.txt to clients currently subscribed to channel 4")
		os.Exit(0)
	}

	var mode string = os.Args[1]
	var channel int8
	var filepath string
	//Determinar el modo seleccionado por el cliente
	switch mode {
	case "receive":
		//Leer canal y, opcionalmente, path de descarga de archivos
		channel = parseChannel(os.Args[3])
		var downloadPath string
		if len(os.Args) == 6 {
			downloadPath = parseDownloadPath(os.Args[5])
		} else {
			downloadPath = DEFAULT_DOWNLOAD_PATH
		}

		subscribeToChannel(channel, downloadPath)
	case "send":
		//Leer canal y path del archivo a enviar
		channel = parseChannel(os.Args[4])
		filepath = os.Args[2]

		sendFileThroughChannel(channel, filepath)
	}
}

func parseChannel(channelStr string) int8 {
	//Conversión del canal a un entero
	channel, parseError := strconv.Atoi(channelStr)
	if parseError != nil {
		fmt.Println("ERROR: Channel is not a valid number: " + parseError.Error())
		os.Exit(1)
	}
	//Se verifica un canal válido (el valor máximo se verifica al conectarse con el servidor)
	if channel < 1 {
		fmt.Println("ERROR: Channel is outside valid range (min: 1)")
		os.Exit(1)
	}
	return int8(channel)
}

func parseDownloadPath(path string) string {
	//Revisar que el path recibido contenga un separador de directorio al final
	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}
	return path
}
