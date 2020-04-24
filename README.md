# Cloudflare Internship Application: Ping Application

This is a simple application built in Go designed to ping a host via ICMP. This program supports both IPv4 and IPv6. It has options for Time To Live and the interval between sending packets.

The original prompt can be found here: [https://github.com/cloudflare-internship-2020/internship-application-systems](https://github.com/cloudflare-internship-2020/internship-application-systems)

## Usage

Make sure to have needed dependencies installed ( `net` from the Golang supplemental libraries):

`go get`

To build:
`go build ping.go`

To run:
`sudo ./ping.go <options> [hostname or ip address]`

Must be run by super user due to raw socket connection required for ICMP.
  
After exiting, the application will display number of packets sent, number of packets recived, the calculated packet loss and the total time taken.


## Command Line Options 

- -interval integer  
  - Set interval between ping requests in seconds (default 1)  
- -ttl integer  
  - Set TTL of ping requests (default 64)  
  
Note that options must go before the target to be pinged
  

## Examples

`sudo ./ping cloudflare.com`  
`sudo ./ping -interval 5 1.1.1.1` for 5 seconds between requests  
`sudo ./ping -ttl 6 2606:4700:4700::1111` for a packet TTL value of 6