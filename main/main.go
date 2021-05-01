/*
packet = トランスポート層でのデータ
ソケット = アプリケーション層とトランスポート槽との架け橋
アプリケーション層でのヘッダーは自由に決められるが、それ以下はソケットがいろいろしてくれる　＝　パケットのヘッダー情報などは見れない
raw ソケット = ネットワークパケットをダイレクトに受け取る・送信することができる = パケットのヘッダー情報が見れる
*/

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
)

type Type uint8

const (
	ECHO_REPLY   Type = 0
	ECHO_REQUEST Type = 8
)

type Echo struct {
	Type     Type
	Code     uint8
	Checksum uint16
	ID       uint16
	Seq      uint16
	Data     []byte
}

func calcChecksum(b []byte) uint16 {
	var checksum uint32 = 0

	for i := 0; i < len(b)-1; i += 2 {
		checksum += uint32(b[i])<<8 | uint32(b[i+1])
	}

	if len(b)&1 != 0 {
		checksum += uint32(b[len(b)-1]) << 8
	}

	// あふれ
	for (checksum >> 16) > 0 {
		checksum = (checksum & 0xffff) + (checksum >> 16)
	}

	return ^(uint16(checksum))
}

// ネットワークバイトオーダーのバイト列に変換する
func (echo *Echo) Mershall() []byte {
	b := make([]byte, 8+len(echo.Data))
	// ビッグエンディアンで並べていく
	b[0] = byte(echo.Type)
	b[1] = byte(echo.Code)
	b[2] = 0
	b[3] = 0
	binary.BigEndian.PutUint16(b[4:6], echo.ID)
	binary.BigEndian.PutUint16(b[6:8], echo.Seq)
	copy(b[8:], echo.Data)

	checksum := calcChecksum(b)
	binary.BigEndian.PutUint16(b[2:4], checksum)

	return b
}

func parsePacket(b []byte) Echo {
	//IPヘッダ: version, headerLen, ・・・
	ihl := int(b[0]&0x0f) * 4
	b = b[ihl:]

	var icmp Echo

	icmp.Type = Type(b[0])
	icmp.Code = uint8(b[1])
	icmp.Checksum = binary.LittleEndian.Uint16(b[2:4])
	icmp.ID = binary.LittleEndian.Uint16(b[4:6])
	icmp.Seq = binary.LittleEndian.Uint16(b[6:8])
	icmp.Data = b[8:]

	return icmp
}

func receivePing(conn *icmp.PacketConn, n int) {
	b := make([]byte, n)
	_, _, err := conn.ReadFrom(b)
	if err != nil {
		log.Fatalf("receive error")
	}

	icmp := parsePacket(b)
	fmt.Println(icmp)
}

func sendPing(conn *icmp.PacketConn, ip *net.IPAddr, seq uint16) int {
	now, _ := time.Now().MarshalBinary()

	echo := Echo{
		Type:     ECHO_REQUEST,
		Code:     0,
		Checksum: 0,
		ID:       uint16(os.Getpid() & 0xffff),
		Seq:      seq,
		Data:     now,
	}
	b := echo.Mershall()
	fmt.Println(b)

	_, err := conn.WriteTo(b, &net.IPAddr{IP: ip.IP})
	if err != nil {
		log.Fatalf("send error")
	}

	return len(b)
}

func ping(ip *net.IPAddr) {
	conn, err := icmp.ListenPacket("ip4:1", "0.0.0.0")
	if err != nil {
		fmt.Println("Socket Create Error")
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	defer conn.Close()

	var seq uint16 = 0
	for ; ; seq += 1 {
		n := sendPing(conn, ip, seq)
		receivePing(conn, n)
		time.Sleep(time.Second / 10)

	}
}

func main() {
	flag.Parse()
	host := flag.Args()[0]
	fmt.Println(host)
	ip, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		log.Fatalf("IP Resolve Error")
	}
	fmt.Println(ip)
	ping(ip)
}
