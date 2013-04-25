package main

import (
	"../statsd"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type RequestLog struct {
	Url         string  `json:"url"`
	ProcessTime float64 `json:"process_time"`
	CreateTime  int64   `json:"create_time"`
}

const maxCount = 100000

func main() {
	client := statsd.NewStatsdClient("127.0.0.1", 8125)
	defer client.Close()
	inputFile, inputError := os.Open("data/requestlog_vip18_vipacceleratecore.log.20130420")
	if inputError != nil {
		fmt.Printf("an error accurred on opening the input file.")
		return
	}
	defer inputFile.Close()
	inputReader := bufio.NewReader(inputFile)
	for i := 0; i < maxCount; i++ {
		inputString, readerError := inputReader.ReadBytes('\n')
		if readerError == io.EOF {
			return
		}
		var msg RequestLog
		err := json.Unmarshal(inputString, &msg)
		if err != nil {
			fmt.Println("error: ", err)
		}
		// fmt.Println("msg: ", msg)
		bucket := "vipservicecore"
		bucket += strings.Replace(msg.Url[:len(msg.Url)-1], "/", ".", -1)
		client.Timing(bucket, int64(msg.ProcessTime*1000))
		fmt.Println("timing:", bucket, int64(msg.ProcessTime*1000))
		time.Sleep(100 * time.Millisecond)
	}
}
