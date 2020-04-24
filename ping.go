/*Written by Ethan Hampton <ethohampton@gmail.com>, April 2020 as a part of my Cloudflare Internship Application
 *The original prompt can be seen at https://github.com/cloudflare-internship-2020/internship-application-systems
 */
package main

import (
	"encoding/binary"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	delay int64 // in nanoseconds
}

func main() {
	log.SetFlags(0) //remove timecodes from loggin

	ttl := flag.Int("ttl", defaultTTL, "Set TTL of ping requests")
	interval := flag.Int("interval", defaultInterval, "Set interval between ping requests in seconds")
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

	log.Printf("PINGing address %s", dest.IP.String())

	//figure out if ipv6 or ipv4
	isIPv6 := dest.IP.To4() == nil

	connString := "ip4:icmp"
	if isIPv6 {
		connString = "ip6:ipv6-icmp"
	}
	//create connection to send and receive on
	c, err := icmp.ListenPacket(connString, "")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	//set ttl for connection
	if isIPv6 {
		c.IPv6PacketConn().SetHopLimit(*ttl)
	} else {
		c.IPv4PacketConn().SetTTL(*ttl)
	}

	//start listening
	listenOutput := make(chan ReturnData)
	go receive(c, isIPv6, listenOutput)

	//used to send packets at a constant interval
	//have to work around weird golang time syntax for multipling an interval by a constants
	tick := time.Tick(time.Duration(*interval) * time.Second)

	//watch for quit events to print out session stats
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	startTime := time.Now()
	seq := 1
	messagesRecived := 0
	for {
		select {
		case <-quit:
			log.Printf("\n\nTranmited %d packets, %d received, %0.1f%% packet loss, total time %dms",
				seq-1, messagesRecived, (float32(seq-1-messagesRecived)/float32(seq-1))*100, time.Now().Sub(startTime).Nanoseconds()/nanoToMilli)
			return
		case data := <-listenOutput:
			if data.ID == -1 {
				log.Println("Time to live exceeded")
			} else {
				log.Printf("Repy received: icmp_seq=%d time=%0.2f ms", data.ID, float64(data.delay)/float64(nanoToMilli))
			}
			messagesRecived++
			break
		case <-tick:
			send(c, &dest, seq, isIPv6)
			seq++
			break
		}
	}
}

//a goroutine to receive ICMP packets and output basic information via the output ReturnData channel
func receive(c *icmp.PacketConn, isIPv6 bool, output chan ReturnData) {
	var proto int
	if isIPv6 {
		proto = ipv6MagicNumber
	} else {
		proto = ipv4MagicNumber
	}
	rb := make([]byte, maxPacketSize)

	for {
		n, _, err := c.ReadFrom(rb)
		//this happens when quiting program just catch it here, otherwise must actually be error
		if err != nil && strings.Contains(err.Error(), "closed network connection") {
			return
		} else if err != nil {
			log.Fatal(err)
		}

		rm, err := icmp.ParseMessage(proto, rb[:n])
		if err != nil {
			log.Fatal(err)
		}

		//Reciving pretty much the same message for ipv4 or ipv6
		//so can cast body to type Echo in both cases which works fine
		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			fallthrough
		case ipv6.ICMPTypeEchoReply:
			returnBody, _ := rm.Body.(*icmp.Echo)
			returnTimeNano := (time.Now().UnixNano() - int64(binary.LittleEndian.Uint64(returnBody.Data)))
			output <- ReturnData{ID: returnBody.Seq, delay: returnTimeNano}

		//handle time exceeded messages
		case ipv4.ICMPTypeTimeExceeded:
			fallthrough
		case ipv6.ICMPTypeTimeExceeded:
			output <- ReturnData{ID: -1}
		default:
			//ignore other messages
		}
	}
}

//send a echo packet to the destination with a given sequence number, encoding the current time in the body
func send(c *icmp.PacketConn, dest net.Addr, seqNum int, isIPv6 bool) {
	t := time.Now().UnixNano()
	body := make([]byte, 8)
	binary.LittleEndian.PutUint64(body, uint64(t))

	var mesType icmp.Type
	if isIPv6 {
		mesType = ipv6.ICMPTypeEchoRequest
	} else {
		mesType = ipv4.ICMPTypeEcho
	}

	wm := icmp.Message{
		Type: mesType, Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  seqNum,
			Data: body,
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
