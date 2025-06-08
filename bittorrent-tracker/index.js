import Server from './server.js'

const server = new Server({
  udp: true,
  http: true
})

server.on('error', function (err) {
  console.log(err.message)
})

server.on('warning', function (err) {
  console.log(err.message)
})

server.on('listening', function () {
  // HTTP
  const httpAddr = server.http.address()
  const httpHost = httpAddr.address !== '::' ? httpAddr.address : 'localhost'
  const httpPort = httpAddr.port
  console.log(`HTTP tracker: http://${httpHost}:${httpPort}/announce`)

  // UDP
  const udpAddr = server.udp.address()
  const udpHost = udpAddr.address
  const udpPort = udpAddr.port
  console.log(`UDP tracker: udp://${udpHost}:${udpPort}`)
})

const port = {http: 6969, udp: 6969}
const hostname = {http: "192.168.16.75", udp: "localhost"}
server.listen(port, hostname, () => {
  console.log(`Tracker server listening on http://${hostname.http}:${port.http}`)
})