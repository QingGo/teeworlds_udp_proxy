package main

import (
	"fmt"
	"net"
	"testing"
)

// simulating message pass with client->proxy->server
func Test_main(t *testing.T) {
	maxClient := 12
	manager := new(UDPClientManager)
	manager.Init(33333, "127.0.0.1", 33334, maxClient)
	go manager.Run()

	serverAddr := net.UDPAddr{
		Port: 33334,
		IP:   net.ParseIP("0.0.0.0"),
	}
	serverSocket, err := net.ListenUDP("udp", &serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSocket.Close()
	message := "Hi there!\n"

	for i := 0; i < 10; i++ {
		go func() {
			conn, err := net.Dial("udp", "127.0.0.1:33333")
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			if _, err := fmt.Fprintf(conn, message); err != nil {
				t.Fatal(err)
			}
		}()

		buf := make([]byte, 32768)
		n, _, err := serverSocket.ReadFromUDP(buf)

		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(string(buf[:n]))
		if msg := string(buf[:n]); msg != message {
			t.Fatalf("Unexpected message:\nGot:\t\t%s\nExpected:\t%s\n", msg, message)
		}
		// Done
	}
	return
}
