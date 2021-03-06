package g2s

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPublish1(t *testing.T) {
	mock := NewMockStatsd(t, 12345)
	defer mock.Shutdown()

	sd, err := NewStatsd("localhost:12345", 0)
	if err != nil {
		t.Fatalf("%s", err)
	}

	sd.IncrementCounter("foo", 1)
	time.Sleep(25 * time.Millisecond)

	if expected, got := 1, mock.Lines(); expected != got {
		t.Errorf("expected %d, got %d", expected, got)
	}
}

func TestPublishMany(t *testing.T) {
	mock := NewMockStatsd(t, 12345)
	defer mock.Shutdown()

	sd, err := NewStatsd("localhost:12345", 0)
	if err != nil {
		t.Fatalf("%s", err)
	}

	sd.IncrementCounter("foo", 1)
	sd.SendSampledTiming("bar", 201, 0.1)
	sd.UpdateGauge("baz", "green")
	time.Sleep(25 * time.Millisecond)

	if expected, got := 3, mock.Lines(); expected != got {
		t.Errorf("expected %d, got %d", expected, got)
	}
}

func TestLoad(t *testing.T) {
	mock := NewMockStatsd(t, 12345)
	defer mock.Shutdown()

	sd, err := NewStatsd("localhost:12345", 0)
	if err != nil {
		t.Fatalf("%s", err)
	}

	sends := 1234 // careful: too high, and we take too long to send
	for i := 0; i < sends; i++ {
		bucket := fmt.Sprintf("bucket-%02d", i%23)
		n := rand.Intn(100)
		sd.SendTiming(bucket, n)
	}
	time.Sleep(50 * time.Millisecond)

	if expected, got := sends, mock.Lines(); expected != got {
		t.Errorf("expected %d, got %d", expected, got)
	}

}

//
//
//

type MockStatsd struct {
	t     *testing.T
	port  int
	lines int
	mtx   sync.Mutex
	ln    *net.UDPConn
	done  chan bool
}

func NewMockStatsd(t *testing.T, port int) *MockStatsd {
	m := &MockStatsd{
		t:    t,
		port: port,
		done: make(chan bool, 1),
	}
	go m.loop()
	time.Sleep(25 * time.Millisecond) // for travis
	return m
}

func (m *MockStatsd) Lines() int {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.lines
}

func (m *MockStatsd) Shutdown() {
	m.ln.Close()
	<-m.done
}

func (m *MockStatsd) loop() {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%d", m.port))
	if err != nil {
		panic(err)
	}

	ln, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	m.ln = ln
	b := make([]byte, 1024*1024)
	defer func() { m.done <- true }()

	for {
		n, _, err := m.ln.ReadFrom(b)
		if err != nil {
			m.t.Logf("Mock Statsd: read error: %s", err)
			return
		}
		s := strings.TrimSpace(string(b[:n]))
		m.t.Logf("Mock Statsd: read %dB: %s", n, s)
		m.mtx.Lock()
		m.lines += len(strings.Split(s, "\n"))
		m.mtx.Unlock()
	}
}
