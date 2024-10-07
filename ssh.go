package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
)

// Valori di default
var (
	defaultPort = "2222"
	username    = "test"
	password    = "password"
)

// Genera le chiavi private e pubbliche per il server SSH
func generateSSHKeys() {
	if _, err := os.Stat("id_rsa"); os.IsNotExist(err) {
		cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-f", "id_rsa", "-N", "")
		err := cmd.Run()
		if err != nil {
			log.Fatalf("Errore durante la generazione delle chiavi: %v", err)
		}
	}
}

// Carica la chiave privata del server SSH
func loadPrivateKey() (ssh.Signer, error) {
	key, err := os.ReadFile("id_rsa")
	if err != nil {
		return nil, err
	}

	privateKey, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// Gestisce il port forwarding (direct-tcpip)
func handleDirectTCPIP(newChannel ssh.NewChannel) {
	type directTCPIP struct {
		DestAddr   string
		DestPort   uint32
		OriginAddr string
		OriginPort uint32
	}
	var channelData directTCPIP
	if err := ssh.Unmarshal(newChannel.ExtraData(), &channelData); err != nil {
		newChannel.Reject(ssh.Prohibited, "failed to parse direct-tcpip")
		return
	}

	// Connessione alla destinazione di inoltro
	remote, err := net.Dial("tcp", fmt.Sprintf("%s:%d", channelData.DestAddr, channelData.DestPort))
	if err != nil {
		newChannel.Reject(ssh.ConnectionFailed, fmt.Sprintf("connection to %s:%d failed: %v", channelData.DestAddr, channelData.DestPort, err))
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Errore nell'accettare il canale: %v", err)
		remote.Close()
		return
	}

	go ssh.DiscardRequests(requests)
	go func() {
		defer channel.Close()
		defer remote.Close()
		go io.Copy(channel, remote)
		io.Copy(remote, channel)
	}()
}

// Gestisce la sessione SSH
func handleSession(newChannel ssh.NewChannel) {
	switch newChannel.ChannelType() {
	case "session":
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Errore nell'accettare il canale: %v", err)
			return
		}
		defer channel.Close()

		// Gestisce richieste tipo shell
		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "shell":
					req.Reply(true, nil)
				default:
					req.Reply(false, nil)
				}
			}
		}(requests)

		// Mantiene la sessione aperta
		io.Copy(channel, channel)

	case "direct-tcpip":
		handleDirectTCPIP(newChannel)

	default:
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unsupported channel type: %s", newChannel.ChannelType()))
	}
}

// Imposta il server SSH
func setupSSHServer(port string) {
	generateSSHKeys()

	privateKey, err := loadPrivateKey()
	if err != nil {
		log.Fatalf("Errore nel caricamento della chiave privata: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == username && string(pass) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for user %s", c.User())
		},
	}
	config.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Errore nell'ascolto sulla porta %s: %v", port, err)
	}
	log.Printf("Server SSH in ascolto sulla porta %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Errore nell'accettare la connessione: %v", err)
			continue
		}

		go func() {
			sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
			if err != nil {
				log.Printf("Errore nel handshake SSH: %v", err)
				return
			}
			defer sshConn.Close()

			go ssh.DiscardRequests(reqs)

			for newChannel := range chans {
				go handleSession(newChannel)
			}
		}()
	}
}

func main() {
	port := flag.String("port", defaultPort, "Porta su cui il server SSH deve essere in ascolto")
	flag.Parse()

	setupSSHServer(*port)
}
