// <<<<<reverseassign,10,1,10,15,pass
package main

import "fmt"

func f() string {
	return "hello"
}
func main() {
var a,b = 3,f()
fmt.Println("The values of a and b are : ",a,b)
}
