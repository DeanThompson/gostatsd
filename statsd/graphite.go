package statsd

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"regexp"
	"time"
)

// Regular expressions used for bucket name normalization
var (
	regSpaces  = regexp.MustCompile("\\s+")
	regSlashes = regexp.MustCompile("\\/")
	regInvalid = regexp.MustCompile("[^a-zA-Z_\\-0-9\\.]")
)

// normalizeBucketName cleans up a bucket name by replacing or translating invalid characters
func normalizeBucketName(name string) string {
	nospaces := regSpaces.ReplaceAllString(name, "_")
	noslashes := regSlashes.ReplaceAllString(nospaces, "-")
	return regInvalid.ReplaceAllString(noslashes, "")
}

// GraphiteClient is an object that is used to send messages to a Graphite server's UDP interface
type GraphiteClient struct {
	conn *net.Conn
	addr string
}

// SendMetrics sends the metrics in a MetricsMap to the Graphite server
func (client *GraphiteClient) SendMetrics(metrics MetricMap) (err error) {
	buf := new(bytes.Buffer)
	now := time.Now().Unix()
	for k, v := range metrics {
		nk := normalizeBucketName(k)
		fmt.Fprintf(buf, "%s %f %d\n", nk, v, now)
	}
	if client.conn != nil {
		_, err = buf.WriteTo(*client.conn)
		if err != nil {
			client.Reconnect()
			return err
		}
	} else {
		client.Reconnect()
		return errors.New("graphite not connected")
	}
	return nil
}

// NewGraphiteClient constructs a GraphiteClient object by connecting to an address
func NewGraphiteClient(addr string) (client GraphiteClient, err error) {
	conn, err := Connect(addr)
	client = GraphiteClient{&conn, addr}
	return
}

func Connect(addr string) (conn net.Conn, err error) {
	conn, err = net.Dial("tcp", addr)
	return conn, err
}

func (client *GraphiteClient) Reconnect() {
	if client.conn != nil {
		err := (*client.conn).Close()
		if err != nil {
			fmt.Println("error closing graphite connection")
			return
		}
		client.conn = nil
	}

	conn, err := Connect(client.addr)
	if err != nil {
		fmt.Println("error reconnect graphite connection")
		return
	}
	client.conn = &conn
}
