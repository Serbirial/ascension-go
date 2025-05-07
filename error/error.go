package error

import "fmt"

func ErrorCheckPanic(e error) {
	if e != nil {
		fmt.Println("\n\nNew Error:")
		fmt.Println(e)
		recover()
	}
}
