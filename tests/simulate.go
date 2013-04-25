package main

import (
	"../statsd"
	"bufio"
	// "encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxCount = 100000

func main() {
	client := statsd.NewStatsdClient("127.0.0.1", 8125)
	defer client.Close()
	inputFile, inputError := os.Open("data/data.txt")
	if inputError != nil {
		fmt.Printf("an error accurred on opening the input file.")
		return
	}
	defer inputFile.Close()
	startTime := time.Now()

	inputReader := bufio.NewReader(inputFile)
	for i := 0; i < maxCount; i++ {
		inputString, readerError := inputReader.ReadBytes('\n')
		if readerError == io.EOF {
			endTime := time.Now()
			seconds := endTime.Sub(startTime).Seconds()
			fmt.Println("processing within ", seconds, "seconds. rate: ", maxCount/seconds)
			return
		}

		columns := strings.Split(strings.TrimSpace(string(inputString)), "\t")
		bucket := columns[0]
		processTime, err := strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			fmt.Println("error reading record", err)
			break
		}
		client.Timing(bucket, processTime)
		// client.Increment(bucket)
		fmt.Println("timing:", bucket, processTime)
		// time.Sleep(1 * time.Millisecond)
	}
}
