package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	WarningLog *log.Logger
	InfoLog    *log.Logger
	ErrorLog   *log.Logger
)

// Errorf logs a formatted error message
func Errorf(format string, v ...interface{}) {
	ErrorLog.Printf(format, v...)
}

// Error logs error messages
func Error(v ...any) {
	ErrorLog.Println(v...)
}

// Infof logs a formatted informational message
func Infof(format string, v ...interface{}) {
	InfoLog.Printf(format, v...)
}

// Info logs informational messages
func Info(v ...any) {
	InfoLog.Println(v...)
}

// Warnf logs a formatted warning message
func Warnf(format string, v ...interface{}) {
	WarningLog.Printf(format, v...)
}

// Warn logs warning messages
func Warn(v ...any) {
	WarningLog.Println(v...)
}

// Fatal logs an error message and terminates the program
func Fatal(v ...any) {
	ErrorLog.Fatal(v...)
}

var logFileName = filepath.Join(os.TempDir(), "claudesquad.log")

var globalLogFile *os.File

// Initialize should be called once at the beginning of the program to set up logging.
// defer Close() after calling this function. It sets the go log output to the file in
// the os temp directory.
func Initialize() {
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("could not open log file: %s", err))
	}

	// Set log format to include timestamp and file/line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	InfoLog = log.New(f, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLog = log.New(f, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(f, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	globalLogFile = f
}

// Close flushes and closes the log file
func Close() {
	_ = globalLogFile.Close()
	// TODO: maybe only print if verbose flag is set?
	fmt.Println("wrote logs to " + logFileName)
}
