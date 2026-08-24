from pathlib import Path
import subprocess

node = Path("p2p/node.go")
text = node.read_text()
old = '''func (n *Node) storePeer(peer *PeerConnection) bool {
\tif peer == nil || peer.Info.NodeID == "" || peer.Info.NodeID == n.NodeID {
\t\treturn false
\t}

\tn.mu.Lock()
\tdefer n.mu.Unlock()
\tif _, exists := n.peers[peer.Info.NodeID]; exists {
\t\treturn false
\t}
\tif !n.canStorePeerLocked(peer) {
\t\treturn false
\t}
\tn.peers[peer.Info.NodeID] = peer
\treturn true
}
'''
new = '''func (n *Node) storePeer(peer *PeerConnection) bool {
\tif peer == nil || peer.Info.NodeID == "" || peer.Info.NodeID == n.NodeID {
\t\treturn false
\t}

\tn.mu.Lock()
\tif _, exists := n.peers[peer.Info.NodeID]; exists {
\t\tn.mu.Unlock()
\t\treturn false
\t}
\tif !n.canStorePeerLocked(peer) {
\t\tn.mu.Unlock()
\t\treturn false
\t}
\tn.peers[peer.Info.NodeID] = peer
\tn.mu.Unlock()

\t// Keep established peers active at the application-protocol level. The
\t// read loop intentionally has an idle deadline, so without a heartbeat two
\t// healthy but quiet nodes would eventually disconnect.
\tgo n.keepPeerAlive(peer)
\treturn true
}
'''
if old not in text:
    raise SystemExit("storePeer target block not found")
node.write_text(text.replace(old, new, 1))

Path("p2p/peer_keepalive.go").write_text('''package p2p

import (
\t"sync/atomic"
\t"time"
)

// PeerKeepaliveInterval is deliberately shorter than PeerReadIdleTimeout so
// quiet but healthy peers exchange application-level traffic before either
// read loop reaches its idle deadline.
const PeerKeepaliveInterval = PeerReadIdleTimeout / 3

var peerKeepaliveNonce atomic.Uint64

func (n *Node) keepPeerAlive(peer *PeerConnection) {
\tn.keepPeerAliveWithInterval(peer, PeerKeepaliveInterval)
}

func (n *Node) keepPeerAliveWithInterval(peer *PeerConnection, interval time.Duration) {
\tif n == nil || peer == nil || interval <= 0 {
\t\treturn
\t}

\tticker := time.NewTicker(interval)
\tdefer ticker.Stop()

\tfor range ticker.C {
\t\tif !n.isCurrentPeerConnection(peer) {
\t\t\treturn
\t\t}
\n\t\tping, err := NewPingMessage(peerKeepaliveNonce.Add(1))
\t\tif err != nil {
\t\t\treturn
\t\t}
\t\tif err := peer.write(ping); err != nil {
\t\t\tn.removePeerConnection(peer)
\t\t\tif peer.conn != nil {
\t\t\t\t_ = peer.conn.Close()
\t\t\t}
\t\t\treturn
\t\t}
\t}
}

func (n *Node) isCurrentPeerConnection(peer *PeerConnection) bool {
\tif n == nil || peer == nil || peer.Info.NodeID == "" {
\t\treturn false
\t}
\tn.mu.RLock()
\tdefer n.mu.RUnlock()
\tcurrent, ok := n.peers[peer.Info.NodeID]
\treturn ok && current == peer
}
''')

Path("p2p/peer_keepalive_test.go").write_text('''package p2p

import (
\t"bufio"
\t"net"
\t"testing"
\t"time"
)

func TestPeerKeepaliveSendsPingAndStopsWhenPeerRemoved(t *testing.T) {
\tnode, err := NewNode("local", "127.0.0.1:0", 0, "tip")
\tif err != nil {
\t\tt.Fatal(err)
\t}

\tlocal, remote := net.Pipe()
\tdefer local.Close()
\tdefer remote.Close()

\tpeer := &PeerConnection{
\t\tInfo:   PeerInfo{NodeID: "remote", ListenAddress: "127.0.0.1:1", TotalWork: "1"},
\t\tconn:   local,
\t\treader: bufio.NewReader(local),
\t}
\tnode.mu.Lock()
\tnode.peers[peer.Info.NodeID] = peer
\tnode.mu.Unlock()

\tdone := make(chan struct{})
\tgo func() {
\t\tnode.keepPeerAliveWithInterval(peer, 5*time.Millisecond)
\t\tclose(done)
\t}()

\tif err := remote.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
\t\tt.Fatal(err)
\t}
\tdata, err := readBoundedPeerMessage(bufio.NewReader(remote))
\tif err != nil {
\t\tt.Fatalf("read keepalive ping: %v", err)
\t}
\tmessage, err := DecodeMessage(data)
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tping, err := DecodePing(message)
\tif err != nil {
\t\tt.Fatalf("keepalive was not a valid ping: %v", err)
\t}
\tif ping.Nonce == 0 {
\t\tt.Fatal("keepalive ping nonce must be non-zero")
\t}

\tnode.removePeerConnection(peer)
\tselect {
\tcase <-done:
\tcase <-time.After(250 * time.Millisecond):
\t\tt.Fatal("keepalive loop did not stop after peer removal")
\t}
}

func TestPeerKeepaliveIntervalPrecedesReadIdleTimeout(t *testing.T) {
\tif PeerKeepaliveInterval <= 0 || PeerKeepaliveInterval >= PeerReadIdleTimeout {
\t\tt.Fatalf("invalid keepalive interval %s for idle timeout %s", PeerKeepaliveInterval, PeerReadIdleTimeout)
\t}
}
''')

subprocess.run(["gofmt", "-w", "p2p/node.go", "p2p/peer_keepalive.go", "p2p/peer_keepalive_test.go"], check=True)
subprocess.run(["go", "test", "./p2p", "-count=1"], check=True)

# Restore the normal CI workflow and remove all one-shot patch helpers before
# committing so only product code/tests remain on the feature branch.
subprocess.run(["git", "checkout", "origin/main", "--", ".github/workflows/ci.yml"], check=True)
for path in [Path(".github/workflows/step65-autopatch.yml"), Path("scripts/step65-autopatch.py")]:
    if path.exists():
        path.unlink()

subprocess.run(["git", "config", "user.name", "github-actions[bot]"], check=True)
subprocess.run(["git", "config", "user.email", "41898282+github-actions[bot]@users.noreply.github.com"], check=True)
subprocess.run(["git", "add", "-A"], check=True)
subprocess.run(["git", "commit", "-m", "Keep quiet P2P peers alive with protocol pings"], check=True)
subprocess.run(["git", "push", "origin", "HEAD:feature/step65-peer-keepalive"], check=True)
