package main

import (
	"encoding/binary"
	"flag"
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

const defaultTTL = 64
const defaultInterval = 1

//ReturnData data passed from ICMP listener to output
type ReturnData struct {
	ID    int
	delay int64 // in nanoseconds for future use
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds) //TODO remove time code from logging
	ttl := flag.Int("ttl", defaultTTL, "Set TTL of ping requests")
	interval := flag.Int("interval", defaultInterval, "Set interval between ping requests in seconds")
	debug := flag.Bool("debug", false, "TODO just for testing")
	flag.Parse()

	rawDestination := flag.Arg(0)
	if rawDestination == "" {
		log.Fatal("usage error: Destination address required")
	}

	var dest net.IPAddr

	//now figure out if destination is a host that needs resolving or a straight ip
	hostIPs, err := net.LookupHost(rawDestination)
	if err != nil {
		log.Fatal(err)
	} else {
		rawIP := net.ParseIP(hostIPs[0])
		if rawIP.String() != "" {
			dest = net.IPAddr{IP: rawIP}
		} else {
			log.Fatal("Error parsing address")
		}
	}

	log.Printf("Using address: %s", dest.IP.String())

	//figure out if ipv6 or ipv4
	isIPv6 := dest.IP.To4() == nil

	connString := "ip4:icmp"
	if isIPv6 {
		connString = "ip6:ipv6-icmp"
	}

	c, err := icmp.ListenPacket(connString, "")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if isIPv6 {
		c.IPv6PacketConn().SetHopLimit(*ttl)
	} else {
		c.IPv4PacketConn().SetTTL(*ttl)
	}

	//TODO TODO FOR TESTING
	if *debug {
		return
	}

	//start of actual technical stuff TODO
	listenOutput := make(chan ReturnData)
	go receive(c, isIPv6, listenOutput)

	seq := 1
	//have to work around weird golang time syntax for multipling an interval by a constants
	tick := time.Tick(time.Duration(*interval) * time.Second)

	for {
		select {
		case data := <-listenOutput:
			log.Printf("Time between ping: %d", data.delay/nanoToMilli)
			break
		case <-tick:
			send(c, &dest, seq, isIPv6)
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
		} //TODO accept expired requests
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
