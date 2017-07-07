package forward

import (
	"context"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/takashabe/go-metrics/collect"
)

func createTestUDPServer(t *testing.T) net.PacketConn {
	server, err := net.ListenPacket("udp", ":0")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	return server
}

func createDummyNetWriterWithKeys(t *testing.T, addr string, keys ...string) *NetWriter {
	sc := collect.NewSimpleCollector()
	for _, key := range keys {
		sc.Add(key, 1)
	}
	mw, err := NewNetWriter(sc, addr)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	nw, ok := mw.(*NetWriter)
	if !ok {
		t.Fatalf("want type *NetWriter, got %v", reflect.TypeOf(mw))
	}
	return nw
}

func TestNetWriter(t *testing.T) {
	ts := createTestUDPServer(t)
	defer ts.Close()

	client := createDummyNetWriterWithKeys(t, ts.LocalAddr().String(), []string{"a", "b", "c"}...)
	if err := client.AddMetrics(client.Source.GetMetricsKeys()...); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if err := client.Flush(); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	expect := `{"a":1.0,"b":1.0,"c":1.0}`

	bs := make([]byte, 1024)
	n, _, err := ts.ReadFrom(bs)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	message := string(bs[:n])
	if !reflect.DeepEqual(message, expect) {
		t.Errorf("want write data %s, got %s", expect, message)
	}
}

func TestNetStream(t *testing.T) {
	ts := createTestUDPServer(t)
	defer ts.Close()

	expect := `{"a":1.0,"b":1.0,"c":1.0}`
	client := createDummyNetWriterWithKeys(t, ts.LocalAddr().String(), []string{"a", "b", "c"}...)
	if err := client.AddMetrics(client.Source.GetMetricsKeys()...); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	client.Interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	client.RunStream(ctx)
	time.Sleep(20 * time.Millisecond) // wait waiting at goroutine
	cancel()

	bs := make([]byte, 1024)
	n, _, err := ts.ReadFrom(bs)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	message := string(bs[:n])
	if !strings.HasPrefix(message, expect) {
		t.Errorf("want has prefix %s, got %s", expect, message[:len(expect)])
	}
}
