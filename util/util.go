package util

import "log"

// Log provides a basic wrapper to format log output.
func Log(key string, value interface{}) {
	log.Printf(`{"%s": "%+v"}`, key, value)
}
