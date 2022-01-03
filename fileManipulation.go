package main

//Archivo que contiene funciones relacionadas con el envío y recepción de archivos a través de TCP

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func receiveFile(connection net.Conn) {
	var exitStatus int = -1 //Código que indica el resultado de procesar la conexión actual
	//Asegurarse de que la conexión se cierre
	defer connection.Close()
	//Leer la longitud del contenido
	var lengthBuffer []byte = make([]byte, 8)
	_, lengthError := connection.Read(lengthBuffer)
	//Error check
	if lengthError != nil {
		fmt.Println("ERROR: Error while reading message's content length: " + lengthError.Error())
		connection.Write([]byte("length read error"))
		exitStatus = 2
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Leer el nombre del archivo
	var filenameBuffer []byte = make([]byte, FILENAME_MAX_LENGTH)
	_, filenameError := connection.Read(filenameBuffer)
	//Error check
	if filenameError != nil {
		fmt.Println("ERROR: Error while reading file name: " + filenameError.Error())
		connection.Write([]byte("filename read error"))
		exitStatus = 2
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Parsear la longitud del contenido
	var contentLength int64
	contentLength = int64(binary.LittleEndian.Uint64(lengthBuffer))
	//Comprobar que la longitud sea válida
	if contentLength <= FILENAME_MAX_LENGTH {
		fmt.Println("ERROR: The client's message specified an invalid content length")
		connection.Write([]byte("invalid content length"))
		exitStatus = 3
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Parsear el nombre del archivo
	var filename string = strings.Split(string(filenameBuffer), "\x00")[0]
	//Comprobar que el nombre del archivo no esté vacío
	if len(filename) == 0 {
		fmt.Println("ERROR: The client's message specified an empty file name")
		connection.Write([]byte("empty filename"))
		exitStatus = 3
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Ya se tiene el nombre del archivo, se muestra un mensaje
	fmt.Println("Receiving file", filename, "from server...")
	//Leer el resto del mensaje (contenido del archivo)
	var fileContentBuffer []byte = make([]byte, contentLength-FILENAME_MAX_LENGTH)
	n, fileContentError := connection.Read(fileContentBuffer)
	//Error check
	if fileContentError != nil {
		fmt.Println("ERROR: Error while reading file content: " + fileContentError.Error())
		connection.Write([]byte("file read error"))
		exitStatus = 2
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	if int64(n) != contentLength-FILENAME_MAX_LENGTH {
		fmt.Printf("ERROR: Could not read file content completely (expected: %d, real: %d)\n", contentLength-FILENAME_MAX_LENGTH, n)
		connection.Write([]byte("file incomplete read"))
		exitStatus = 2
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Se guarda el archivo en el equipo
	var file *os.File
	var fileError error
	file, fileError = os.Create(RECEIVED_FILES_PATH + filename)
	defer file.Close()
	//Error check
	if fileError != nil {
		fmt.Println("ERROR: Error while creating received file in filesystem: " + fileError.Error())
		exitStatus = 5
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Volcar contenido en el archivo creado
	var fileBuffer *bytes.Buffer = bytes.NewBuffer(fileContentBuffer)
	var fileSize int64
	var copyError error
	fileSize, copyError = io.Copy(file, fileBuffer)
	//Error check
	if copyError != nil {
		fmt.Println("ERROR: Error while copying file to buffer: " + copyError.Error())
		exitStatus = 5
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Ya se descargó el archivo
	fmt.Printf("File %v received (%d bytes)\n", filename, fileSize)
	connection.Write([]byte("received"))
	exitStatus = 0
	fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
}

func sendFile(messageHeader []byte, filename []byte, file *os.File) {
	//Asegurarse de que el archivo se cierre
	defer file.Close()
	//Copiar el contenido del archivo en cuestión a un buffer
	var fileBuffer *bytes.Buffer = bytes.NewBuffer(nil)
	var fileSize int64
	var copyError error
	fileSize, copyError = io.Copy(fileBuffer, file)
	//Error check
	if copyError != nil {
		fmt.Println("ERROR: Error while copying file to buffer: " + copyError.Error())
		os.Exit(5)
	}
	//Completar el mensaje
	var message, fileContent, lengthBuffer []byte
	fileContent = fileBuffer.Bytes()
	message = append(message, messageHeader...)
	//Calcular la longitud del contendido (nombre + contenido del archivo)
	var contentLength int64 = FILENAME_MAX_LENGTH + fileSize
	lengthBuffer = make([]byte, 8)
	binary.LittleEndian.PutUint64(lengthBuffer, uint64(contentLength))
	//Añadir la longitud al mensaje
	message = append(message, lengthBuffer...)
	//Añadir el nombre del archivo al mensaje
	message = append(message, filename...)
	//Si el nombre del archivo no ocupaba el tamaño máximo, es necesario llenar los espacios faltantes
	if FILENAME_MAX_LENGTH > len(filename) {
		message = append(message, []byte(strings.Repeat("\x00", FILENAME_MAX_LENGTH-len(filename)))...)
	}
	//Añadir el contenido del archivo al mensaje
	message = append(message, fileContent...)

	//Verificar la longitud del mensaje
	if int64(len(message)) != 10+contentLength {
		fmt.Printf("ERROR: Error while creating message (expected length: %d, real length: %d)\n", 10+contentLength, len(message))
		os.Exit(3)
	}

	//Iniciar conexión con el servidor para enviar el mensaje
	fmt.Println("Connecting to server...")
	var connection net.Conn
	var connectionError error
	connection, connectionError = net.Dial("tcp", "127.0.0.1:"+SERVER_PORT)

	if connectionError != nil {
		fmt.Println("ERROR: Error while connecting to server: " + connectionError.Error())
		os.Exit(2)
	}
	//Asegurarse de que la conexión se cierre
	defer connection.Close()
	//Enviar el mensaje con el archivo en cuestión
	var messageError error
	_, messageError = connection.Write(message)
	if messageError != nil {
		fmt.Println("ERROR: Error while sending message to server: " + messageError.Error())
		os.Exit(2)
	}

	//Obtener respuesta del servidor
	fmt.Println("File sent. Awaiting server response...")
	var responseBuffer []byte = make([]byte, BUFFER_SIZE)
	n, responseError := connection.Read(responseBuffer)

	if responseError != nil {
		fmt.Println("ERROR: Error while getting server's response: " + responseError.Error())
		os.Exit(2)
	}

	//Parsear respuesta
	var response string = string(responseBuffer[:n])

	//Interpretar respuesta
	switch response {
	case "received":
		fmt.Println("Server received file successfully. It will be sent to all subscribed clients on selected channel.")
	default:
		fmt.Println("ERROR: Server error (" + response + ")")
		os.Exit(2)
	}
}
