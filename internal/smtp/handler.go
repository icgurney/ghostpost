package smtp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"ghostpost/internal/storage"

	"github.com/google/uuid"
	"github.com/pires/go-proxyproto"
)

const maxEmailSize = 10 * 1024 * 1024 // 10MB limit

type Handler struct {
	conn          net.Conn
	reader        *bufio.Reader
	emailData     strings.Builder
	inDataPhase   bool
	currentSize   int
	storage       storage.Storage
	commands      map[string]func(string)
	acceptDomains []string
}

func NewHandler(conn net.Conn, storage storage.Storage, acceptDomains []string) *Handler {
	h := &Handler{
		conn:          conn,
		reader:        bufio.NewReader(conn),
		storage:       storage,
		acceptDomains: acceptDomains,
	}

	// Map commands to handler methods
	h.commands = map[string]func(string){
		"HELO": h.handleHelo,
		"EHLO": h.handleHelo,
		"MAIL": h.handleMailFrom,
		"RCPT": h.handleRcptTo,
		"DATA": h.handleData,
		"QUIT": h.handleQuit,
	}

	return h
}

func (h *Handler) Handle() {
	defer h.conn.Close()

	// Get client info from proxy protocol
	if proxyConn, ok := h.conn.(*proxyproto.Conn); ok {
		header := proxyConn.ProxyHeader()
		if header != nil {
			log.Printf("New connection from %s", proxyConn.RemoteAddr().String())
		}
	}

	// Send greeting
	h.sendResponse("220 Ready to receive mail\r\n")

	for {
		line, err := h.reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading: %v", err)
			}
			return
		}

		fmt.Printf("<- %s", line)

		if h.inDataPhase {
			if strings.TrimSpace(line) == "." {
				h.handleEndOfData()
			} else {
				// handle dot-stuffing
				if strings.HasPrefix(line, "..") {
					line = line[1:]
				}

				h.currentSize += len(line)
				if h.currentSize > maxEmailSize {
					h.sendResponse("552 Message size exceeds maximum permitted\r\n")
					h.conn.Close()
					return
				}
				h.emailData.WriteString(line)
			}
			continue
		}

		h.handleCommand(line)
	}
}

func (h *Handler) handleEndOfData() {
	fmt.Println("\nReceived complete email. Parsing...")

	fmt.Printf("\n=== Email Contents ===\n")
	fmt.Printf("%s", h.emailData.String())
	fmt.Printf("==================\n\n")

	// Save email
	id := uuid.NewString()
	err := h.storage.SaveEmail(context.Background(), id, strings.NewReader(h.emailData.String()))
	if err != nil {
		log.Printf("Failed to save email: %v", err)
		h.sendResponse("450 Requested mail action not taken: mailbox unavailable\r\n")
		return
	}

	h.sendResponse("250 Ok\r\n")
	h.inDataPhase = false
	h.emailData.Reset()
}

func (h *Handler) handleCommand(line string) {
	// Remove trailing white space escape characters /r/n
	line = strings.TrimSpace(line)
	cmd := strings.SplitN(strings.ToUpper(line), " ", 2)[0]
	handler, ok := h.commands[cmd]
	if !ok {
		h.sendResponse("500 Command not recognized\r\n")
		return
	}
	handler(line)
}

// Command handlers
func (h *Handler) handleHelo(line string) {
	h.sendResponse("250 Ok\r\n")
}

func (h *Handler) handleMailFrom(line string) {
	h.sendResponse("250 Ok\r\n")
}

func (h *Handler) handleRcptTo(line string) {
	// Extract email from "RCPT TO:<email@domain.com>"
	start := strings.Index(line, "<")
	end := strings.Index(line, ">")
	if start == -1 || end == -1 {
		h.sendResponse("501 Syntax error in parameters\r\n")
		return
	}

	email := line[start+1 : end]
	for _, domain := range h.acceptDomains {
		if strings.HasSuffix(email, "@"+domain) {
			h.sendResponse("250 Ok\r\n")
			return
		}
	}
	h.sendResponse("550 Relay not permitted\r\n")
}

func (h *Handler) handleData(line string) {
	h.inDataPhase = true
	h.sendResponse("354 Start mail input; end with <CRLF>.<CRLF>\r\n")
}

func (h *Handler) handleQuit(line string) {
	h.sendResponse("221 Bye\r\n")
}

func (h *Handler) sendResponse(response string) {
	fmt.Printf("-> %s", response)
	h.conn.Write([]byte(response))
}
