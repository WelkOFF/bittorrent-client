package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

import bencode "github.com/jackpal/bencode-go"

type torrentInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Name        string `bencode:"name"`
	Length      int    `bencode:"length"`
}

type TrackerServerResponseBodyRaw struct {
	Peers    string
	Interval string
}

type torrentRaw struct {
	Announce     string
	Comment      string
	CreationDate int64       `bencode:"creation date"`
	Info         torrentInfo `bencode:"info"`
}

type Peer struct {
	IP   net.IP
	Port uint16
}

// Unmarshal parses peer IP addresses and ports from a buffer
func Unmarshal(peersBin []byte) ([]Peer, error) {
	const peerSize = 6 // 4 for IP, 2 for port
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		err := fmt.Errorf("Received malformed peers")
		return nil, err
	}
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
	}
	return peers, nil
}

func (i *torrentInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (t *torrentRaw) buildTrackerUrl(port uint16) (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		log.Fatalln("Invalid Tracker URL ", err)
	}

	infoHash, err := t.Info.hash()
	if err != nil {
		log.Fatalln("Info hash failed ", err)
	}

	params := url.Values{
		"info_hash":  []string{string(infoHash[:])},
		"peer_id":    []string{"-TR2940-k8hj0wgej6ch"},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Info.Length)},
	}
	base.RawQuery = params.Encode()
	fmt.Println(base)
	return "", nil
}

func main() {

	fmt.Println("Hello kur!")
	file, err := os.Open("ubuntu-23.10.1-desktop-amd64.iso.torrent")
	defer file.Close()
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(file.Name())

	torrentRaw := torrentRaw{}

	err = bencode.Unmarshal(file, &torrentRaw)
	fmt.Println(torrentRaw.Announce)

	if err != nil {
		log.Fatalln("Could not parse torrent file")
	}

	url, _ := torrentRaw.buildTrackerUrl(6881)

	trackerURL := torrentRaw.Announce
	resp, err := http.Get(trackerURL)
	if err != nil {
		log.Fatalln("Could not fetch tracker list ", err)
	}

	fmt.Println(resp.Status, url)

	var respBodyBytes bytes.Buffer
	resp.Write(&respBodyBytes)

	respData := TrackerServerResponseBodyRaw{}
	err = bencode.Unmarshal(resp.Body, &respData)

	fmt.Println(respData)
}
