// <<<<< reverseassign,11,3,11,24,pass
package main

import "fmt"

func f() (float64, float64) {
	return 1.4, 2.3
}

func main() {
  var i, x float64 = f()
  fmt.Println(i, x)
}
