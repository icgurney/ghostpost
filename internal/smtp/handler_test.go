package smtp

import (
	"io"
	"net"
	"testing"
	"time"

	"ghostpost/internal/storage/mock"
)

type mockConn struct {
	net.Conn
	readData  string
	writeData string
}

func (m *mockConn) Read(p []byte) (n int, err error) {
	if m.readData == "" {
		return 0, io.EOF
	}
	n = copy(p, m.readData)
	m.readData = m.readData[n:]
	return n, nil
}

func (m *mockConn) Write(p []byte) (n int, err error) {
	m.writeData += string(p)
	return len(p), nil
}

func (m *mockConn) Close() error { return nil }

func TestHandler_HandleEmail(t *testing.T) {
	conn := &mockConn{
		readData: "HELO localhost\r\n" +
			"MAIL FROM:<sender@example.com>\r\n" +
			"RCPT TO:<recipient@example.com>\r\n" +
			"DATA\r\n" +
			"From: sender@example.com\r\n" +
			"To: recipient@example.com\r\n" +
			"Subject: Test Email\r\n\r\n" +
			"This is a test email.\r\n" +
			".\r\n" +
			"QUIT\r\n",
	}

	store := mock.NewStorage()
	acceptDomains := []string{"ghostpost.sh", "xn--9q8hgh.ws"}

	handler := NewHandler(conn, store, acceptDomains)
	go handler.Handle()

	time.Sleep(100 * time.Millisecond)

	if len(store.SavedEmails) != 1 {
		t.Errorf("Expected 1 saved email, got %d", len(store.SavedEmails))
	}
}
