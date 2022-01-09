package main

//Archivo que contiene funciones relacionadas a la interacción con el servidor en los dos modos del cliente

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	filepath2 "path/filepath"
	"syscall"
)

//Función para enviar una solicitud de suscripción a un determinado canal al servidor
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

	//El cliente se comunica con el servidor para suscribirse al canal (enviando un mensaje)
	var message []byte
	var command int8 = 0
	var addressBuffer []byte = []byte(clientAddress)
	var contentLength int64 = int64(len(addressBuffer))
	//Se genera el mensaje como tal
	message = createSimpleMessage(command, channel, addressBuffer)

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
	//Recibir respuesta del servidor (leyendo primero el header)
	var headerBuffer []byte = make([]byte, 10)
	var responseCommand int8
	var responseContentLength int64
	_, headerError := connection.Read(headerBuffer)
	//Error check
	if headerError != nil {
		fmt.Println("ERROR: Error while getting server's response header: " + headerError.Error())
		connection.Close()
		os.Exit(2)
	}
	//Parsear header (comando, longitud del contenido)
	responseCommand = int8(headerBuffer[0])
	responseContentLength = int64(binary.LittleEndian.Uint64(headerBuffer[2:]))
	//Leer respuesta
	var contentBuffer []byte = make([]byte, responseContentLength)
	var content string
	_, contentError := connection.Read(contentBuffer)
	//Error check
	if contentError != nil {
		fmt.Println("ERROR: Error while getting server's response content: " + contentError.Error())
		connection.Close()
		os.Exit(2)
	}
	//Parsear respuesta
	content = string(contentBuffer)

	//Cerrar conexión, pues ya se obtuvo una respuesta
	connection.Close()

	//Interpretar respuesta
	switch responseCommand {
	case 2:
		fmt.Println("Client successfully subscribed to channel", channel)
		fmt.Println("Awaiting incoming file transfers on " + clientAddress + "...")
	case 3:
		fmt.Println("ERROR: Server error (" + content + ")")
		os.Exit(2)
	default:
		fmt.Println("Invalid command received from server:", responseCommand)
		os.Exit(2)
	}

	//Una vez exitosa la suscripción, el cliente queda esperando transferencias de archivos mediante el listener
	//Primero dejemos corriendo una goroutine para cancelar la suscripción al canal si el programa termina
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM) //El programa responderá a las señales SIGINT y SIGTERM
	go func() {
		<-sig
		unsubscribe(channel, addressBuffer)
		os.Exit(0)
	}()
	//Ahora se atienden las transferencias
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
		go receiveFile(incomingConnection, downloadPath, channel)
	}
}

//Función para enviar una solicitud de envío de archivo a un determinado canal al servidor
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

func unsubscribe(channel int8, address []byte) {
	//Anunciar que el cliente va a cancelar su suscripción al canal
	fmt.Printf("\nCancelando suscripción de %v al canal %d...\n", string(address), channel)
	//Armar el mensaje a enviar al servidor
	var message []byte = createSimpleMessage(4, channel, address)
	//Conectarse al servidor para enviar el mensaje
	connection, connError := net.Dial("tcp", "127.0.0.1:"+SERVER_PORT)
	if connError != nil {
		fmt.Println("ERROR: Error while connecting to server: " + connError.Error())
		os.Exit(2)
	}
	//Asegurarse de cerrar la conexión al salir
	defer connection.Close()
	//Enviar mensaje
	_, sendError := connection.Write(message)
	if sendError != nil {
		fmt.Println("ERROR: Error while sending message to server: " + sendError.Error())
		os.Exit(2)
	}
	fmt.Println("Request sent. Awaiting server response...")
	//Recibir respuesta del servidor (leyendo primero el header)
	var headerBuffer []byte = make([]byte, 10)
	var responseCommand int8
	var responseContentLength int64
	_, headerError := connection.Read(headerBuffer)
	//Error check
	if headerError != nil {
		fmt.Println("ERROR: Error while getting server's response header: " + headerError.Error())
		os.Exit(2)
	}
	//Parsear header (comando, longitud del contenido)
	responseCommand = int8(headerBuffer[0])
	responseContentLength = int64(binary.LittleEndian.Uint64(headerBuffer[2:]))
	//Leer respuesta
	var contentBuffer []byte = make([]byte, responseContentLength)
	var content string
	_, contentError := connection.Read(contentBuffer)
	//Error check
	if contentError != nil {
		fmt.Println("ERROR: Error while getting server's response content: " + contentError.Error())
		os.Exit(2)
	}
	//Parsear respuesta
	content = string(contentBuffer)

	//Interpretar respuesta
	switch responseCommand {
	case 2:
		fmt.Println("Client successfully unsubscribed from channel", channel)
	case 3:
		fmt.Println("ERROR: Server error (" + content + ")")
		os.Exit(2)
	default:
		fmt.Println("Invalid command received from server:", responseCommand)
		os.Exit(2)
	}
}
