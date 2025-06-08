import bencode from 'bencode'
import Debug from 'debug'
import dgram from 'dgram'
import EventEmitter from 'events'
import http from 'http'
import series from 'run-series'
import string2compact from 'string2compact'
import { hex2bin } from 'uint8-util'

import common from './lib/common.js'
import Swarm from './lib/swarm.js'
import parseHttpRequest from './lib/parse-http.js'
import parseUdpRequest from './lib/parse-udp.js'

const debug = Debug('bittorrent-tracker:server')

/**
 * BitTorrent tracker server.
 *
 * HTTP service which responds to GET requests from torrent clients. Requests include
 * metrics from clients that help the tracker keep overall statistics about the torrent.
 * Responses include a peer list that helps the client participate in the torrent.
 *
 * @param {Object}  opts                options object
 * @param {Number}  opts.interval       tell clients to announce on this interval (ms)
 * @param {boolean|Object} opts.http    start an http server?, or options for http.createServer (default: true)
 * @param {boolean|Object} opts.udp     start a udp server?, or extra options for dgram.createSocket (default: true)
 */
class Server extends EventEmitter {
  constructor (opts = {}) {
    super()
    debug('new server %s', JSON.stringify(opts))

    this.intervalMs = opts.interval
      ? opts.interval
      : 10 * 60 * 1000 // 10 min


    this.peersCacheLength = opts.peersCacheLength
    this.peersCacheTtl = opts.peersCacheTtl

    this._listenCalled = false
    this.listening = false
    this.destroyed = false
    this.torrents = {}

    this.http = null
    this.udp4 = null
    this.udp6 = null

    // start an http tracker unless the user explictly says no
    if (opts.http !== false) {
      this.http = http.createServer(isObject(opts.http) ? opts.http : undefined)
      this.http.on('error', err => { this._onError(err) })
      this.http.on('listening', onListening)

      // Add default http request handler on next tick to give user the chance to add
      // their own handler first. Handle requests untouched by user's handler.
      process.nextTick(() => {
        this.http.on('request', (req, res) => {
          if (res.headersSent) return
          this.onHttpRequest(req, res)
        })
      })
    }

    // start a udp tracker unless the user explicitly says no
    if (opts.udp !== false) {
      this.udp4 = this.udp = dgram.createSocket({
        type: 'udp4',
        reuseAddr: true,
        ...(isObject(opts.udp) ? opts.udp : undefined)
      })
      this.udp4.on('message', (msg, rinfo) => { this.onUdpRequest(msg, rinfo) })
      this.udp4.on('error', err => { this._onError(err) })
      this.udp4.on('listening', onListening)

      this.udp6 = dgram.createSocket({
        type: 'udp6',
        reuseAddr: true,
        ...(isObject(opts.udp) ? opts.udp : undefined)
      })
      this.udp6.on('message', (msg, rinfo) => { this.onUdpRequest(msg, rinfo) })
      this.udp6.on('error', err => { this._onError(err) })
      this.udp6.on('listening', onListening)
    }

    let num = !!this.http + !!this.udp4 + !!this.udp6
    const self = this
    function onListening () {
      num -= 1
      if (num === 0) {
        self.listening = true
        debug('listening')
        self.emit('listening')
      }
    }
  }

  _onError (err) {
    this.emit('error', err)
  }

  listen (...args) /* port, hostname, onlistening */{
    if (this._listenCalled || this.listening) throw new Error('server already listening')
    this._listenCalled = true

    const lastArg = args[args.length - 1]
    if (typeof lastArg === 'function') this.once('listening', lastArg)

    const port = toNumber(args[0]) || args[0] || 0
    const hostname = typeof args[1] !== 'function' ? args[1] : undefined

    debug('listen (port: %o hostname: %o)', port, hostname)

    const httpPort = isObject(port) ? (port.http || 0) : port
    const udpPort = isObject(port) ? (port.udp || 0) : port

    // binding to :: only receives IPv4 connections if the bindv6only sysctl is set 0,
    // which is the default on many operating systems
    const httpHostname = isObject(hostname) ? hostname.http : hostname
    const udp4Hostname = isObject(hostname) ? hostname.udp : hostname
    const udp6Hostname = isObject(hostname) ? hostname.udp6 : hostname

    if (this.http) this.http.listen(httpPort, httpHostname)
    if (this.udp4) this.udp4.bind(udpPort, udp4Hostname)
    if (this.udp6) this.udp6.bind(udpPort, udp6Hostname)
  }

  close (cb = noop) {
    debug('close')

    this.listening = false
    this.destroyed = true

    if (this.udp4) {
      try {
        this.udp4.close()
      } catch (err) {}
    }

    if (this.udp6) {
      try {
        this.udp6.close()
      } catch (err) {}
    }

    if (this.http) this.http.close(cb)
    else cb(null)
  }

  createSwarm (infoHash, cb) {
    if (ArrayBuffer.isView(infoHash)) infoHash = infoHash.toString('hex')

    process.nextTick(() => {
      const swarm = this.torrents[infoHash] = new Server.Swarm(infoHash, this)
      cb(null, swarm)
    })
  }

  getSwarm (infoHash, cb) {
    if (ArrayBuffer.isView(infoHash)) infoHash = infoHash.toString('hex')

    process.nextTick(() => {
      cb(null, this.torrents[infoHash])
    })
  }

  onHttpRequest (req, res, opts = {}) {
    let params
    try {
      params = parseHttpRequest(req, opts)
      params.httpReq = req
      params.httpRes = res
    } catch (err) {
      res.end(bencode.encode({
        'failure reason': err.message
      }))

      // even though it's an error for the client, it's just a warning for the server.
      // don't crash the server because a client sent bad data :)
      this.emit('warning', err)
      return
    }

    this._onRequest(params, (err, response) => {
      if (err) {
        this.emit('warning', err)
        response = {
          'failure reason': err.message
        }
      }
      if (this.destroyed) return res.end()

      delete response.action // only needed for UDP encoding
      res.end(bencode.encode(response))

      if (params.action === common.ACTIONS.ANNOUNCE) {
        this.emit(common.EVENT_NAMES[params.event], params.addr, params)
      }
    })
  }

  onUdpRequest (msg, rinfo) {
    let params
    try {
      params = parseUdpRequest(msg, rinfo)
    } catch (err) {
      this.emit('warning', err)
      // Do not reply for parsing errors
      return
    }

    this._onRequest(params, (err, response) => {
      if (err) {
        this.emit('warning', err)
        response = {
          action: common.ACTIONS.ERROR,
          'failure reason': err.message
        }
      }
      if (this.destroyed) return

      response.transactionId = params.transactionId
      response.connectionId = params.connectionId

      const buf = makeUdpPacket(response)

      try {
        const udp = (rinfo.family === 'IPv4') ? this.udp4 : this.udp6
        udp.send(buf, 0, buf.length, rinfo.port, rinfo.address)
      } catch (err) {
        this.emit('warning', err)
      }

      if (params.action === common.ACTIONS.ANNOUNCE) {
        this.emit(common.EVENT_NAMES[params.event], params.addr, params)
      }
    })
  }

  _onRequest (params, cb) {
    if (params && params.action === common.ACTIONS.CONNECT) {
      cb(null, { action: common.ACTIONS.CONNECT })
    } else if (params && params.action === common.ACTIONS.ANNOUNCE) {
      this._onAnnounce(params, cb)
    } else if (params && params.action === common.ACTIONS.SCRAPE) {
      this._onScrape(params, cb)
    } else {
      cb(new Error('Invalid action'))
    }
  }

  _onAnnounce (params, cb) {
    const self = this

    getOrCreateSwarm((err, swarm) => {
      if (err) return cb(err)
      announce(swarm)
    })

    // Get existing swarm, or create one if one does not exist
    function getOrCreateSwarm (cb) {
      self.getSwarm(params.info_hash, (err, swarm) => {
        if (err) return cb(err)
        if (swarm) return cb(null, swarm)
        self.createSwarm(params.info_hash, (err, swarm) => {
          if (err) return cb(err)
          cb(null, swarm)
        })
      })
    }

    function announce (swarm) {
      if (!params.event || params.event === 'empty') params.event = 'update'
      swarm.announce(params, (err, response) => {
        if (err) return cb(err)

        if (!response.action) response.action = common.ACTIONS.ANNOUNCE
        if (!response.interval) response.interval = Math.ceil(self.intervalMs / 1000)

        if (params.compact === 1) {
          const peers = response.peers

          // Find IPv4 peers
          response.peers = string2compact(peers.filter(peer => common.IPV4_RE.test(peer.ip)).map(peer => `${peer.ip}:${peer.port}`))
          // Find IPv6 peers
          response.peers6 = string2compact(peers.filter(peer => common.IPV6_RE.test(peer.ip)).map(peer => `[${peer.ip}]:${peer.port}`))
        } else if (params.compact === 0) {
          // IPv6 peers are not separate for non-compact responses
          response.peers = response.peers.map(peer => ({
            'peer id': hex2bin(peer.peerId),
            ip: peer.ip,
            port: peer.port
          }))
        }

        cb(null, response)
      })
    }
  }

  _onScrape (params, cb) {
    if (params.info_hash == null) {
      // if info_hash param is omitted, stats for all torrents are returned
      // TODO: make this configurable!
      params.info_hash = Object.keys(this.torrents)
    }

    series(params.info_hash.map(infoHash => cb => {
      this.getSwarm(infoHash, (err, swarm) => {
        if (err) return cb(err)
        if (swarm) {
          swarm.scrape(params, (err, scrapeInfo) => {
            if (err) return cb(err)
            cb(null, {
              infoHash,
              complete: (scrapeInfo && scrapeInfo.complete) || 0,
              incomplete: (scrapeInfo && scrapeInfo.incomplete) || 0
            })
          })
        } else {
          cb(null, { infoHash, complete: 0, incomplete: 0 })
        }
      })
    }), (err, results) => {
      if (err) return cb(err)

      const response = {
        action: common.ACTIONS.SCRAPE,
        files: {},
        flags: { min_request_interval: Math.ceil(this.intervalMs / 1000) }
      }

      results.forEach(result => {
        response.files[hex2bin(result.infoHash)] = {
          complete: result.complete || 0,
          incomplete: result.incomplete || 0,
          downloaded: result.complete || 0 // TODO: this only provides a lower-bound
        }
      })

      cb(null, response)
    })
  }
}

Server.Swarm = Swarm

function makeUdpPacket (params) {
  let packet
  switch (params.action) {
    case common.ACTIONS.CONNECT: {
      packet = Buffer.concat([
        common.toUInt32(common.ACTIONS.CONNECT),
        common.toUInt32(params.transactionId),
        params.connectionId
      ])
      break
    }
    case common.ACTIONS.ANNOUNCE: {
      packet = Buffer.concat([
        common.toUInt32(common.ACTIONS.ANNOUNCE),
        common.toUInt32(params.transactionId),
        common.toUInt32(params.interval),
        common.toUInt32(params.incomplete),
        common.toUInt32(params.complete),
        params.peers
      ])
      break
    }
    case common.ACTIONS.SCRAPE: {
      const scrapeResponse = [
        common.toUInt32(common.ACTIONS.SCRAPE),
        common.toUInt32(params.transactionId)
      ]
      for (const infoHash in params.files) {
        const file = params.files[infoHash]
        scrapeResponse.push(
          common.toUInt32(file.complete),
          common.toUInt32(file.downloaded), // TODO: this only provides a lower-bound
          common.toUInt32(file.incomplete)
        )
      }
      packet = Buffer.concat(scrapeResponse)
      break
    }
    case common.ACTIONS.ERROR: {
      packet = Buffer.concat([
        common.toUInt32(common.ACTIONS.ERROR),
        common.toUInt32(params.transactionId || 0),
        Buffer.from(String(params['failure reason']))
      ])
      break
    }
    default:
      throw new Error(`Action not implemented: ${params.action}`)
  }
  return packet
}

function isObject (obj) {
  return typeof obj === 'object' && obj !== null
}

function toNumber (x) {
  x = Number(x)
  return x >= 0 ? x : false
}

function noop () {}

export default Server
