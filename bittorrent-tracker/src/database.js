import sqlite3 from 'sqlite3'
import path from 'path'
import { fileURLToPath } from 'url'

// Helper to get DB path relative to this file
const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const DB_PATH = path.join(__dirname, 'peers.sqlite3')

class PeerDB {
  constructor() {
    this.db = new sqlite3.Database(DB_PATH)
    this.db.serialize(() => {
      this.db.run(`CREATE TABLE IF NOT EXISTS peers (
        info_hash TEXT,
        peer_id TEXT,
        ip TEXT,
        port INTEGER,
        complete INTEGER,
        PRIMARY KEY (info_hash, peer_id)
      )`)
    })
  }

  upsertPeer(infoHash, peer) {
    this.db.run(
      `INSERT INTO peers (info_hash, peer_id, ip, port, complete) VALUES (?, ?, ?, ?, ?)
       ON CONFLICT(info_hash, peer_id) DO UPDATE SET ip=excluded.ip, port=excluded.port, complete=excluded.complete`,
      [infoHash, peer.peerId, peer.ip, peer.port, peer.complete ? 1 : 0]
    )
  }

  loadPeers(infoHash, cb) {
    this.db.all(
      `SELECT * FROM peers WHERE info_hash = ?`,
      [infoHash],
      (err, rows) => {
        if (err) return cb(err)
        cb(null, rows)
      }
    )
  }
}

export default PeerDB
