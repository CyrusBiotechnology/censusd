# censusd

censusd is a distributed population counter packaged as a go library. It's used
in [CyrusBiotechnology/dozy](github.com/CyrusBiotechnology/dozy) to support
minimum population counts.

*Note: censusd only supports IPv4 addressing. IPv6 only provides multicast
facilities, but Cyrus Biotechnology deploys primarily to Google Compute Engine.
Compute Engine does not yet support IPv6 OR multicast. Please submit a pull
reque*

# Usage

    import (
      "github.com/CyrusBiotechnology/censusd"
    )

    func main() {
      listen := net.UDPAddr{
        IP:   net.IPv4(0, 0, 0, 0),
        Port: 19091,
      }

      stats, err := censusd.Serve(done, "udp4", &listen)
      if err != nil {
        panic(err)
      }

      for {
        time.Sleep(time.Second*1)
        stats.Mutex.RUnlock()
        println("peers:", stats.Nodes)
        stats.Mutex.RUnlock()
      }
    }