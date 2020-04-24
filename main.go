package main

import (
	"encoding/binary"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"golang.org/x/net/icmp"
)

const maxPacketSize = 1500
const ipv4MagicNumber = 1
const ipv6MagicNumber = 58
const nanoToMilli = 1000000

//ReturnData data passed from ICMP listener to output
type ReturnData struct {
	ID    int
	delay int64 // in nanoseconds for future use
}

func main() {
	/*
		c.IPv4PacketConn().SetTTL(64) // for ipv4
		c.IPv6PacketConn().HopLimit(64) // for ipv6
	*/
	c, err := icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		log.Println("Some sort of error here")
		log.Fatal(err)
	}
	defer c.Close()

	log.Println("Starting receiver!")
	listenOutput := make(chan ReturnData)

	seq := 1
	dest := &net.IPAddr{IP: net.ParseIP("1.1.1.1")}

	//send(c, dest, seq, false)
	//log.Println("Sent first")
	go receive(c, false, listenOutput)

	tick := time.Tick(time.Second)
	var data ReturnData
	for {
		select {
		case data = <-listenOutput:
			log.Printf("Time between ping: %d", data.delay/nanoToMilli)
			break
		case <-tick:
			send(c, dest, seq, false)
			seq++
			break
		}
	}
}

func receive(c *icmp.PacketConn, isIPv6 bool, output chan ReturnData) {
	log.Println("Started receiver")

	rb := make([]byte, maxPacketSize)
	for {
		//read
		log.Println("Listening")
		n, peer, err := c.ReadFrom(rb)
		if err != nil {
			log.Println("Can't read from connection")
			log.Fatal(err)
		}

		rm, err := icmp.ParseMessage(ipv4MagicNumber, rb[:n])
		if err != nil {
			log.Fatal(err)
		}

		//Reciving pretty much the same message so can case body to type echo in both cases which works fine //TODO TESTTHIS
		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			fallthrough
		case ipv6.ICMPTypeEchoReply:
			log.Printf("got reflection from %v\n", peer)

			returnBody, _ := rm.Body.(*icmp.Echo)
			returnTimeNano := (time.Now().UnixNano() - int64(binary.LittleEndian.Uint64(returnBody.Data)))
			output <- ReturnData{ID: returnBody.ID, delay: returnTimeNano}
		default:
			log.Printf("got %+v; want echo reply", rm)
		}
	}
}

func send(c *icmp.PacketConn, dest net.Addr, seqNum int, isIPv6 bool) {

	t := time.Now().UnixNano()
	body := make([]byte, 8)
	binary.LittleEndian.PutUint64(body, uint64(t))

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: seqNum,
			Data: body, //[]byte("TESTING"), //TODO use time sent here, this would help once we got it back in order to determine what time it was sent
			//Also would make it easier to run asynconously as it could just present packets as we get them. Also include sequence #s for tracking !
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := c.WriteTo(wb, dest); err != nil {
		log.Fatal(err)
	}

}
