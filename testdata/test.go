package main

import "fmt"

func main() {
	_, err := fmt.Println("Hello")
	_, err = fmt.Println(" world")
	_, err = fmt.Println(err)
}
