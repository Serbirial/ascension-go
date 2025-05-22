package error

import "log"

// Checks error, logs it, and recovers if possible
func ErrorCheckPanic(e error) {
	if e != nil {
		log.Println("\n\nNew Error:")
		log.Println(e)
		recover()
	}
}
