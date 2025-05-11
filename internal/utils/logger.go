package utils

import (
	"log"
	"os"
)

var (
	// InfoLogger логгер для информационных сообщений
	InfoLogger *log.Logger
	// ErrorLogger логгер для сообщений об ошибках
	ErrorLogger *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// LogInfo логирует информационное сообщение
func LogInfo(format string, v ...interface{}) {
	InfoLogger.Printf(format, v...)
}

// LogError логирует сообщение об ошибке
func LogError(format string, v ...interface{}) {
	ErrorLogger.Printf(format, v...)
}
