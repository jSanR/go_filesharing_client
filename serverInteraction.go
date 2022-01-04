package main

//Archivo que contiene funciones relacionadas a la interacción con el servidor en los dos modos del cliente

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	filepath2 "path/filepath"
)

func subscribeToChannel(channel int8, downloadPath string) {
	//Anunciar el modo en el que se ejecuta el cliente
	fmt.Println("Receive mode: channel", channel)
	//Se crea un listener del cliente para poder recibir mensajes del servidor cuando un archivo sea enviado
	var listener net.Listener
	var listenerError error
	listener, listenerError = net.Listen("tcp", "127.0.0.1:")
	defer listener.Close()
	//Error check
	if listenerError != nil {
		fmt.Println("ERROR: Error while starting client listener for subscription: " + listenerError.Error())
		os.Exit(2)
	}

	//Se obtiene en un string la IP y puerto del listener del cliente
	var clientAddress string = listener.Addr().String()

	//El cliente se comunica con el servidor para suscribirse al canal (enviando un mensaje
	var message []byte
	var command int8 = 0
	//Añadir el comando al mensaje
	message = append(message, byte(command))
	//Añadir el canal al mensaje
	message = append(message, byte(channel))
	//Añadir la longitud del contenido al mensaje
	var lengthBuffer []byte = make([]byte, 8)
	var addressBuffer []byte = []byte(clientAddress)
	var contentLength int64 = int64(len(addressBuffer))
	binary.LittleEndian.PutUint64(lengthBuffer, uint64(contentLength))
	message = append(message, lengthBuffer...)
	//Añadir el contenido (dirección del listener del cliente) al mensaje
	message = append(message, addressBuffer...)

	//Verificar que la longitud del mensaje sea la correcta
	if int64(len(message)) != 10+contentLength {
		fmt.Printf("ERROR: Error while creating subscription message (expected length: %d, real length: %d)\n", 10+contentLength, len(message))
		os.Exit(3)
	}

	//Ahora es posible enviar el mensaje al servidor
	fmt.Println("Sending subscription request to server...")
	//Se entabla la conexión con el servidor
	var connection net.Conn
	var connectionError error
	connection, connectionError = net.Dial("tcp", "127.0.0.1:"+SERVER_PORT)
	//Error check
	if connectionError != nil {
		fmt.Println("ERROR: Error while connecting to server: " + connectionError.Error())
		os.Exit(2)
	}
	//Se envía el mensaje
	_, err := connection.Write(message)
	//Error check
	if err != nil {
		fmt.Println("ERROR: Error while sending message to server: " + err.Error())
		os.Exit(2)
	}

	fmt.Println("Request sent. Awaiting server response...")
	//Recibir respuesta del servidor
	var responseBuffer []byte = make([]byte, BUFFER_SIZE)
	n, responseError := connection.Read(responseBuffer)
	//Error check
	if responseError != nil {
		fmt.Println("ERROR: Error while getting server's response: " + responseError.Error())
		connection.Close()
		os.Exit(2)
	}

	//Parsear respuesta
	var response string = string(responseBuffer[:n])

	//Cerrar conexión, pues ya se obtuvo una respuesta
	connection.Close()

	//Interpretar respuesta
	switch response {
	case "success":
		fmt.Println("Client successfully subscribed to channel", channel)
		fmt.Println("Awaiting incoming file transfers on " + clientAddress + "...")
	default:
		fmt.Println("ERROR: Server error (" + response + ")")
		os.Exit(2)
	}

	//Una vez exitosa la suscripción, el cliente queda esperando transferencias de archivos mediante el listener
	for {
		var incomingConnection net.Conn
		var incomingConnError error
		//Aceptar conexión
		incomingConnection, incomingConnError = listener.Accept()
		//Error check
		if incomingConnError != nil {
			fmt.Println("ERROR: Error while accepting incoming connection: " + incomingConnError.Error())
			os.Exit(3)
		}

		//Recibir el archivo y guardarlo
		go receiveFile(incomingConnection, downloadPath)
	}
}

func sendFileThroughChannel(channel int8, filepath string) {
	//Anunciar el modo en el que se ejecuta el cliente
	fmt.Println("Send mode: file "+filepath+", channel", channel)
	//Se obtiene el nombre del archivo y se revisa su longitud
	var filename string = filepath2.Base(filepath)
	if len([]byte(filename)) > FILENAME_MAX_LENGTH {
		fmt.Println("ERROR: File name is too long (max length including file extension:", FILENAME_MAX_LENGTH, "characters)")
		os.Exit(1)
	}
	//Se crea la cabecera del mensaje que se enviará al servidor (comando, canal)
	var header []byte
	var command int8 = 1
	//Añadir el comando al header
	header = append(header, byte(command))
	//Añadir el canal al header
	header = append(header, byte(channel))

	//Se abre el archivo en cuestión
	var file *os.File
	var fileError error
	file, fileError = os.Open(filepath)
	//Error check
	if fileError != nil {
		fmt.Println("ERROR: Error while opening file: " + fileError.Error())
		os.Exit(5)
	}

	//Se realiza el envío del archivo al servidor
	sendFile(header, []byte(filename), file)
}
