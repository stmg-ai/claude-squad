package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var logFileName = filepath.Join(os.TempDir(), "claudesquad.log")

var globalLogFile *os.File

// Initialize should be called once at the beginning of the program to set up logging.
// defer Close() after calling this function. It sets the go log output to the file in
// the os temp directory.
func Initialize() {
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("could not open logger file: %s", err))
	}

	// Set log format to include timestamp and file/line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	globalLogFile = f
	log.SetOutput(f)
}

func Close() {
	_ = globalLogFile.Close()
	// TODO: maybe only print if verbose flag is set?
	fmt.Println("wrote logs to " + logFileName)
}
