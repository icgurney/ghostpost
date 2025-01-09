package integration

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"ghostpost/internal/smtp"
	"ghostpost/internal/storage/mock"
)

type testEmail struct {
	from    string
	to      string
	subject string
	body    string
}

func waitForServer(t *testing.T, port string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	addr := "localhost:" + port

	for time.Now().Before(deadline) {
		if conn, err := net.Dial("tcp", addr); err == nil {
			conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Server failed to start on port %s within timeout", port)
}

func TestMultipleClientsIntegration(t *testing.T) {
	store := mock.NewStorage()
	acceptDomains := []string{"ghostpost.sh", "xn--9q8hgh.ws"}

	server := smtp.NewServer(":2526", store, acceptDomains)
	go server.ListenAndServe()
	waitForServer(t, "2526")

	// Update test emails to use accepted domains
	emails := []testEmail{
		{
			from:    "sender1@example.com",
			to:      "recipient1@ghostpost.sh",
			subject: "Test Email 1",
			body:    "This is test email 1",
		},
		{
			from:    "sender2@example.com",
			to:      "recipient2@ghostpost.sh",
			subject: "Test Email 2",
			body:    "This is test email 2",
		},
		{
			from:    "sender3@example.com",
			to:      "recipient3@ghostpost.sh",
			subject: "Test Email 3",
			body:    "This is test email 3",
		},
	}

	var wg sync.WaitGroup
	for i, email := range emails {
		wg.Add(1)
		go func(e testEmail, clientID int) {
			defer wg.Done()
			err := sendTestEmail(e, clientID)
			if err != nil {
				t.Errorf("Client %d failed: %v", clientID, err)
			}
		}(email, i)
	}

	wg.Wait()

	// Verify all emails were saved
	if len(store.SavedEmails) != len(emails) {
		t.Errorf("Expected %d saved emails, got %d", len(emails), len(store.SavedEmails))
	}
}

func TestRejectUnacceptedDomain(t *testing.T) {
	store := mock.NewStorage()
	acceptDomains := []string{"ghostpost.sh", "xn--9q8hgh.ws"}

	server := smtp.NewServer(":2527", store, acceptDomains)
	go server.ListenAndServe()
	waitForServer(t, "2527")

	email := testEmail{
		from:    "sender@example.com",
		to:      "recipient@example.com", // Unaccepted domain
		subject: "Test Email",
		body:    "This should be rejected",
	}

	err := sendTestEmail(email, 0)
	if err == nil {
		t.Error("Expected error for unaccepted domain, got nil")
	}

	if !strings.Contains(err.Error(), "550") {
		t.Errorf("Expected 550 error, got: %v", err)
	}

	if len(store.SavedEmails) != 0 {
		t.Errorf("Expected 0 saved emails, got %d", len(store.SavedEmails))
	}
}

func sendTestEmail(email testEmail, clientID int) error {
	conn, err := net.Dial("tcp", "localhost:2526")
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := &smtpTestClient{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		clientID: clientID,
	}

	// SMTP conversation
	if err := client.expectResponse("220"); err != nil {
		return err
	}

	if err := client.sendAndExpect("HELO localhost", "250"); err != nil {
		return err
	}

	if err := client.sendAndExpect(fmt.Sprintf("MAIL FROM:<%s>", email.from), "250"); err != nil {
		return err
	}

	if err := client.sendAndExpect(fmt.Sprintf("RCPT TO:<%s>", email.to), "250"); err != nil {
		return err
	}

	if err := client.sendAndExpect("DATA", "354"); err != nil {
		return err
	}

	// Send email content
	client.send(fmt.Sprintf("From: %s", email.from))
	client.send(fmt.Sprintf("To: %s", email.to))
	client.send(fmt.Sprintf("Subject: %s", email.subject))
	client.send("")
	client.send(email.body)

	if err := client.sendAndExpect(".", "250"); err != nil {
		return err
	}

	if err := client.sendAndExpect("QUIT", "221"); err != nil {
		return err
	}

	return nil
}

type smtpTestClient struct {
	conn     net.Conn
	reader   *bufio.Reader
	clientID int
}

func (c *smtpTestClient) send(line string) error {
	_, err := fmt.Fprintf(c.conn, "%s\r\n", line)
	return err
}

func (c *smtpTestClient) expectResponse(prefix string) error {
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}
	if !strings.HasPrefix(response, prefix) {
		return fmt.Errorf("client %d: expected response prefix %q, got %q", c.clientID, prefix, response)
	}
	return nil
}

func (c *smtpTestClient) sendAndExpect(send, expect string) error {
	if err := c.send(send); err != nil {
		return fmt.Errorf("failed to send %q: %v", send, err)
	}
	return c.expectResponse(expect)
}
