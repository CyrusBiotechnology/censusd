# censusd

censusd is a distributed population counter packaged as a go library. It's used
in [CyrusBiotechnology/dozy](github.com/CyrusBiotechnology/dozy) to support
minimum population counts.

*Note: censusd only supports IPv4 addressing. IPv6 only provides multicast
facilities, but Cyrus Biotechnology deploys primarily to Google Compute Engine.
Compute Engine does not yet support IPv6 OR multicast. Please submit a pull
reque*

# Usage

    package main

    import (
      "github.com/CyrusBiotechnology/censusd"
      "net"
      "time"
    )

    func main() {
      listen := net.UDPAddr{
        IP:   net.IPv4(0, 0, 0, 0),
        Port: 19092,
      }

      done := make(chan struct{})

      // "nodename" may be left blank, in which case the hostname will be used.
      censusServer, err := censusd.NewServer(&listen, "groupname", "nodename")
      if err != nil {
        panic(err)
      }
      stats, err := censusServer.Serve(done, "udp4")
      if err != nil {
        panic(err)
      }

      for {
        time.Sleep(time.Second * 10)
        stats.Mutex.RLock()
        println("peers:", stats.Nodes)
        stats.Mutex.RUnlock()
      }
    }
