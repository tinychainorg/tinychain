// In the olden days of bitcoin, satoshi wrote a hack to use IRC channels as a way to find peers.
// And now, just as Satoshi did, we too shall make a hack
// To use public BitTorrent trackers as a way to find peers.
//
// https://wiki.theory.org/BitTorrent_Tracker_Protocol
// https://www.bittorrent.org/beps/bep_0015.html
package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/jackpal/bencode-go"
)

const trackerList = `https://tracker.tamersunion.org:443/announce
https://tracker.renfei.net:443/announce
https://tracker.loligirl.cn:443/announce
https://tracker.gcrenwp.top:443/announce
https://www.peckservers.com:9443/announce
https://tracker.yemekyedim.com:443/announce
https://tracker.lilithraws.org:443/announce
https://tracker.imgoingto.icu:443/announce
https://tracker.cloudit.top:443/announce
https://t1.hloli.org:443/announce
https://trackers.run:443/announce
https://tracker.pmman.tech:443/announce
https://tracker.ipfsscan.io:443/announce
https://tracker.gbitt.info:443/announce
https://tracker-zhuqiy.dgj055.icu:443/announce`

const (
	trackerURL = "https://t1.hloli.org:443/announce"
	port       = 6881
)

type TrackerResponse struct {
	Interval int            `bencode:"interval"`
	Peers    []TrackerPeers `bencode:"peers"`
}

type TrackerPeers struct {
	IP   string `bencode:"ip"`
	Port int    `bencode:"port"`
	ID   string `bencode:"peer id"`
}

// func generatePeerID() string { // TODO: unused, might use to keep IPv6 + Port
// 	b := make([]byte, 20)
// 	_, err := rand.Read(b)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	return hex.EncodeToString(b)
// }

func addPeerToSwarm(peerID string, infoHash string) error {
	fmt.Println("Adding peer to swarm")
	fmt.Println("Peer ID:", peerID)
	fmt.Println("Infohash:", infoHash)

	params := url.Values{}
	params.Add("info_hash", infoHash)
	params.Add("peer_id", peerID)
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", "0")
	params.Add("event", "started")

	resp, err := http.Get(trackerURL + "?" + params.Encode()) //do we only want to announce to one tracker
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("addPeerToSwarm Tracker response:", string(body))
	return nil
}

func getPeers(infoHash, peerID string) (*TrackerResponse, error) {
	params := url.Values{}
	params.Add("info_hash", infoHash)
	params.Add("peer_id", peerID)
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", "0")

	resp, err := http.Get(trackerURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Println("getPeers Tracker response status:", resp.Status)
	fmt.Println("getPeers Tracker response body:", resp.Body)

	var trackerResp TrackerResponse
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Interval: %d\n", trackerResp.Interval)
	// fmt.Printf("Peers: %s\n", trackerResp.Peers)
	for _, peer := range trackerResp.Peers {
		fmt.Printf("Peer: id=%s addr=%s:%d\n", peer.ID, peer.IP, peer.Port)
	}

	return &trackerResp, nil
}
