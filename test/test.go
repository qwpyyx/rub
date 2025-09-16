package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {
	//str := "{2210274049 莫昌康 09010013 2023-09-17 20:00 03:00 1}"
	dateStr := "08:00"
	var startTime int
	arr := strings.Split(dateStr, ":")
	startTime, _ = strconv.Atoi(arr[0])
	fmt.Println(startTime)
	new1 := strconv.Itoa(startTime + 1)
	if len(new1) == 1 {
		new1 = "0" + new1
	}
	fmt.Println(new1)
}
