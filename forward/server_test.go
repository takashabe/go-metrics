package forward

import (
	"net"
	"reflect"
	"testing"

	"github.com/takashabe/go-metrics/collect"
)

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
	server, err := net.ListenPacket("udp", ":0")
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	defer server.Close()

	client := createDummyNetWriterWithKeys(t, server.LocalAddr().String(), []string{"a", "b", "c"}...)
	if err := client.AddMetrics(client.Source.GetMetricsKeys()...); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if err := client.Flush(); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	expect := []byte(`{"a":1.0,"b":1.0,"c":1.0}`)

	bs := make([]byte, 1024)
	n, _, err := server.ReadFrom(bs)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	message := bs[:n]
	if !reflect.DeepEqual(message, expect) {
		t.Errorf("want write data %s, got %s", expect, message)
	}
}
