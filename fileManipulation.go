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

//Función para recibir un archivo proveniente del servidor
func receiveFile(connection net.Conn, downloadPath string, channel int8) {
	var exitStatus int = -1 //Código que indica el resultado de procesar la conexión actual
	//Asegurarse de que la conexión se cierre
	defer connection.Close()
	//Leer el header del mensaje
	var headerBuffer []byte = make([]byte, 10)
	var headerCommand, headerChannel int8
	var contentLength int64
	_, headerError := connection.Read(headerBuffer)
	//Error check
	if headerError != nil {
		fmt.Println("ERROR: Error while reading message header: " + headerError.Error())
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("header read error")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
		exitStatus = 2
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Parsear el header del mensaje (comando, canal, longitud del contenido)
	headerCommand = int8(headerBuffer[0])
	headerChannel = int8(headerBuffer[1])
	contentLength = int64(binary.LittleEndian.Uint64(headerBuffer[2:]))
	//Comprobar validez de los 3 campos
	//Comando (debe ser el comando send o 1)
	if headerCommand != 1 {
		fmt.Println("ERROR: Invalid command (should have value 1 for \"send\")")
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("invalid command")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
		exitStatus = 3
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Canal (debe ser el mismo que el recibido como parámetro)
	if headerChannel != channel {
		fmt.Println("ERROR: Subscribed and received channels differ")
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("incorrect channel")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
		exitStatus = 3
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Longitud de contenido (debe ser como mayor al tamaño máximo de nombre de archivo)
	if contentLength <= FILENAME_MAX_LENGTH {
		fmt.Println("ERROR: The client's message specified an invalid content length")
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("invalid content length")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
		exitStatus = 3
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Leer el nombre del archivo
	var filenameBuffer []byte = make([]byte, FILENAME_MAX_LENGTH)
	_, filenameError := connection.Read(filenameBuffer)
	//Error check
	if filenameError != nil {
		fmt.Println("ERROR: Error while reading file name: " + filenameError.Error())
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("filename read error")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
		exitStatus = 2
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Parsear el nombre del archivo (bytes no utilizados se llenan con el caracter \x00)
	var filename string = strings.Split(string(filenameBuffer), "\x00")[0]
	//Comprobar que el nombre del archivo no esté vacío
	if len(filename) == 0 {
		fmt.Println("ERROR: The client's message specified an empty file name")
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("empty filename")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
		exitStatus = 3
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Ya se tiene el nombre del archivo, se muestra un mensaje
	fmt.Println("Receiving file", filename, "from server...")
	//Leer el resto del mensaje (contenido del archivo)
	var fileContentBuffer []byte = make([]byte, 0) //Se irá llenando iterativamente a partir de tempBuffer
	var tempBuffer []byte = make([]byte, BUFFER_SIZE)
	var readLength int64 = 0
	for {
		//Leer al buffer temporal
		n, fileContentError := connection.Read(tempBuffer)
		//Error check
		if fileContentError == io.EOF { //Se concluyó la lectura
			break
		} else if fileContentError != nil { //Hubo un error de otro tipo
			fmt.Println("ERROR: Error while reading file content: " + fileContentError.Error())
			_, err := connection.Write(createSimpleMessage(3, channel, []byte("file read error")))
			if err != nil {
				fmt.Println("ERROR: Error while sending response to server: " + err.Error())
			}
			exitStatus = 2
			fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
			return
		}
		//Añadir lo leído al buffer del archivo
		fileContentBuffer = append(fileContentBuffer, tempBuffer...)
		//Actualizar longitud leída
		readLength += int64(n)
		//Si ya se leyó completamente el archivo, se sale del bucle
		if readLength == contentLength-FILENAME_MAX_LENGTH {
			break
		}
	}
	if readLength != contentLength-FILENAME_MAX_LENGTH {
		fmt.Printf("ERROR: Could not read file content completely (expected: %d, real: %d)\n", contentLength-FILENAME_MAX_LENGTH, readLength)
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("file incomplete read")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
		exitStatus = 2
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Se crea un nuevo archivo en el equipo con el nombre del archivo enviado
	var file *os.File
	var fileError error
	file, fileError = os.Create(downloadPath + filename)
	defer file.Close()
	//Error check
	if fileError != nil {
		fmt.Println("ERROR: Error while creating received file in filesystem: " + fileError.Error())
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("file creation failed")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
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
		_, err := connection.Write(createSimpleMessage(3, channel, []byte("file copying failed")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		}
		exitStatus = 5
		fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
		return
	}
	//Ya se descargó el archivo
	fmt.Printf("File %v received (%d bytes)\n", filename, fileSize)
	_, err := connection.Write(createSimpleMessage(2, channel, []byte("received")))
	if err != nil {
		fmt.Println("ERROR: Error while sending response to server: " + err.Error())
		exitStatus = 5
	} else {
		exitStatus = 0
	}
	fmt.Printf("Handled file transfer (status: %d)\n", exitStatus)
}

//Función para enviar un archivo al servidor
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
	//Completar el mensaje (excepto el archivo, pues este se enviará iterativamente luego)
	var message, lengthBuffer []byte
	//Se añade el header al mensaje
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
	//message = append(message, fileContent...)

	//Verificar la longitud del mensaje
	if int64(len(message)) != 10+FILENAME_MAX_LENGTH {
		fmt.Printf("ERROR: Error while creating message (expected length: %d, real length: %d)\n", 10+contentLength, len(message))
		os.Exit(3)
	}

	//Iniciar conexión con el servidor para enviar el mensaje y el archivo
	fmt.Println("Connecting to server...")
	var connection net.Conn
	var connectionError error
	connection, connectionError = net.Dial("tcp", "127.0.0.1:"+SERVER_PORT)
	//Error check
	if connectionError != nil {
		fmt.Println("ERROR: Error while connecting to server: " + connectionError.Error())
		os.Exit(2)
	}
	fmt.Println("Connection successful")
	//Asegurarse de que la conexión se cierre
	defer connection.Close()
	//Enviar el mensaje
	var messageError error
	_, messageError = connection.Write(message)
	//Error check
	if messageError != nil {
		fmt.Println("ERROR: Error while sending message to server: " + messageError.Error())
		os.Exit(2)
	}
	//Enviar el archivo de forma iterativa (con un buffer temporal)
	fmt.Printf("Sending %d bytes...\n", fileSize)
	var tempBuffer []byte = make([]byte, BUFFER_SIZE)
	var sentLength int = 0
	//Leer el archivo desde el inicio nuevamente
	_, seekError := file.Seek(0, io.SeekStart)
	if seekError != nil {
		fmt.Println("ERROR: Error while seeking file to the start: " + seekError.Error())
		os.Exit(5)
	}
	for {
		//Leer del archivo al buffer temporal
		readBytes, readError := file.Read(tempBuffer)
		if readError != nil {
			if readError == io.EOF {
				fmt.Printf("File read completely (sent %d bytes)\n", sentLength)
				break
			}
			fmt.Println("ERROR: Error while reading file contents: " + readError.Error())
			os.Exit(2)
		}
		//fmt.Printf("Read %d bytes | ", readBytes)
		//Enviar el buffer al cliente
		sentBytes, sendError := connection.Write(tempBuffer[:readBytes])
		if sendError != nil {
			fmt.Println("ERROR: Error while sending file contents: " + sendError.Error())
			os.Exit(2)
		}
		//Actualizar la cantidad enviada
		sentLength += sentBytes
		//Comprobar que lo que se lee se esté enviando completamente
		if readBytes != sentBytes {
			fmt.Println("ERROR: File buffer was sent incompletely")
			os.Exit(2)
		}
		//fmt.Printf("Sent %d bytes\n", sentBytes)
	}
	//Asegurarse de que el archivo se leyó y envió completamente
	if int64(sentLength) != fileSize {
		fmt.Println("ERROR: File was sent incompletely")
		os.Exit(2)
	}
	//Obtener respuesta del servidor (empezando por el header)
	fmt.Println("File sent. Awaiting server response...")
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
	//Leer contenido del mensaje
	var contentBuffer []byte = make([]byte, responseContentLength)
	var content string
	_, contentError := connection.Read(contentBuffer)
	if contentError != nil {
		fmt.Println("ERROR: Error while getting server's response content: " + contentError.Error())
		os.Exit(2)
	}
	//Parsear contenido del mensaje
	content = string(contentBuffer)

	//Interpretar respuesta
	switch responseCommand {
	case 2:
		fmt.Println("Server received file successfully. It will be sent to all subscribed clients on selected channel.")
	case 3:
		fmt.Println("ERROR: Server error (" + content + ")")
		os.Exit(2)
	default:
		fmt.Println("ERROR: Invalid command received from server:", responseCommand)
	}
}
