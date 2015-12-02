package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Luzifer/go-openssl"
)

const (
	_echoServerAddr      = "127.0.0.1:62863"
	_expectAESCiphertext = "U2FsdGVkX19KIJ9OQJKT/yHGMrS+5SsBAAjetomptQ0="
	_secret              = "p0S8rX680*48"
)

func servEcho() {
	l, err := net.Listen("tcp", _echoServerAddr)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + _echoServerAddr)
	for {
		// Listen for an incoming connection.
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go func(c net.Conn) {
			defer c.Close()

			_, err := io.Copy(c, c)
			switch err {
			case io.EOF:
				err = nil
				return
			case nil:
				return
			}
			panic(err)
		}(c)
	}
}

func TestMain(m *testing.M) {
	// start echo server
	go servEcho()

	// start listen
	os.Setenv("SECRET", _secret)

	go main()

	rand.Seed(time.Now().UnixNano())

	// TODO: better way to wait for server to start
	time.Sleep(time.Second)
	os.Exit(m.Run())
}

func TestTextDecryptAES(t *testing.T) {
	o := openssl.New()

	dec, err := o.DecryptString(_secret, _expectAESCiphertext)
	if err != nil {
		panic(err)
	}
	if strings.Compare(string(dec), _echoServerAddr) != 0 {
		panic(errors.New("not match"))
	}
}

func encryptText(plaintext, passphrase string) ([]byte, error) {
	o := openssl.New()

	return o.EncryptString(passphrase, plaintext)
}

func randomBytes(n int) []byte {

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i := 0; i < n; i++ {
		b[i] = byte(rand.Int())
	}

	return b
}

func testEchoRound(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(time.Second * 10))

	n := rand.Int()%2048 + 10
	out := randomBytes(n)
	n0, err := conn.Write(out)
	if err != nil {
		panic(err)
	}

	rcv := make([]byte, n)
	n1, err := io.ReadFull(conn, rcv)
	if err != nil && err != io.EOF {
		panic(err)
	}
	if !bytes.Equal(out[:n0], rcv[:n1]) {
		fmt.Println("out: ", n0, "in:", n1)

		fmt.Println("out: ", hex.EncodeToString(out), "in:", hex.EncodeToString(rcv))
		panic(errors.New("echo server reply is not match"))
	}
}

func TestEchoServer(t *testing.T) {
	conn, err := net.Dial("tcp", _echoServerAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	n := rand.Int() % 10
	for i := 0; i < n; i++ {
		testEchoRound(conn)
	}
}

func TestProtocolDecrypt(*testing.T) {
	// * test decryption
	conn, err := net.Dial("tcp", "127.0.0.1:"+_DefaultPort)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	b, err := encryptText(_echoServerAddr, _secret)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write(b)
	if err != nil {
		panic(err)
	}

	_, err = conn.Write([]byte("\n"))
	if err != nil {
		panic(err)
	}

	for i := 0; i < 5; i++ {
		testEchoRound(conn)
	}
}

// TODO: test decryption with extra bytes in packet and check data

// TODO: test decryption with seperated packet simulate loss connection and check data

// TODO: benchmark 100, 1000 connect with 1k 10k 100k 1m data

func BenchmarkEncryptText(b *testing.B) {
	s1 := string(randomBytes(255))
	s2 := string(randomBytes(32))
	for i := 0; i < b.N; i++ {
		_, err := encryptText(s1, s2)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkDecryptText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		o := openssl.New()
		_, err := o.DecryptString(_secret, _expectAESCiphertext)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkEcho(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestEchoServer(&testing.T{})
	}
}

func BenchmarkLatency(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestProtocolDecrypt(&testing.T{})
	}
}

// with echo server with random hanging
// * benchmark latency
// * benchmark throughput
// * benchmark copy-on-write performance BackendAddrCache
// * benchmark memory footprint
