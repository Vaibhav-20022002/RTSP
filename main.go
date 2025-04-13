// This program connects to an RTSP source using the gortsplib library,
// prints the SDP (in JSON format) and metadata about the media tracks,
// and listens for RTP packets. Each received RTP packet is printed in JSON.

// To run this program:
//   go run main.go <rtsp-url>
// For example:
//   go run main.go rtsp://localhost:8554/mystream

package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

func main() {
	// Ensure RTSP URL is provided :
	if len(os.Args) < 2 {
		log.Fatalln("Usage:", os.Args[0], "<rtsp-url>")
	}

	// Parsing RTSP URL :
	rtspURL := os.Args[1]
	parsedURL, err := base.ParseURL(rtspURL)
	if err != nil {
		log.Fatalf("Cannot parse RTSP URL : %v", err)
	}

	log.Println("Starting RTSP client for URL :", rtspURL)

	// Create a new RTSP client with timeouts and enabling any port. :
	// The client will be used to connect, describe, setup, and play the stream.
	client := &gortsplib.Client{
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  5 * time.Second,
		AnyPortEnable: true,
	}

	// ---------------------------------
	// Step 0: CONNECT to the RTSP Server
	// ---------------------------------
	// The client.Start method connects to the RTSP server.
	err = client.Start(parsedURL.Scheme, parsedURL.Host)
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	// Ensure the client connection is closed on exit.
	defer client.Close()

	// ----------------------------
	// Step 1: DESCRIBE Request
	// ----------------------------
	// The DESCRIBE request retrieves the session description (SDP) and media tracks.
	desc, _, err := client.Describe(parsedURL)
	if err != nil {
		log.Fatalf("Error during DESCRIBE: %v", err)
	}

	// Convert the SDP description to JSON format :
	descJSON, err := json.MarshalIndent(desc, "", " ")
	if err != nil {
		log.Printf("Error marshaling SDP description to JSON: %v", err)
	} else {
		log.Println("SDP in JSON:")
		log.Println(string(descJSON))
	}

	// ----------------------------
	// Step 2: SETUP Media
	// ----------------------------
	// Setup all medias :
	err = client.SetupAll(desc.BaseURL, desc.Medias)
	if err != nil {
		log.Printf("Error setting up medias: %v", err)
	}

	// ---------------------------------------
	// Step 3: Register RTP Packet Callback
	// ---------------------------------------
	// The OnPacketRTP callback is called whenever an RTP packet is received :
	client.OnPacketRTPAny(func(medi *description.Media, forma format.Format, pkt *rtp.Packet) {
		packetInfo := map[string]any{
			"version":           pkt.Version,
			"sequence_number":   pkt.SequenceNumber,
			"timestamp":         pkt.Timestamp,
			"extension":         pkt.Extension,
			"padding":           pkt.Padding,
			"marker":            pkt.Marker,
			"payload_type":      pkt.PayloadType,
			"ssrc":              pkt.SSRC,
			"csrc":              pkt.CSRC,
			"extensions":        pkt.Extensions,
			"extension_profile": pkt.ExtensionProfile,
		}

		packetJSON, err := json.MarshalIndent(packetInfo, "", "  ")
		if err != nil {
			log.Printf("Error marshaling RTP packet to JSON: %v", err)
			return
		}
		log.Println("Received RTP packet:")
		log.Println(string(packetJSON))
	})

	// -----------------------------------
	// Step 4: Start the RTSP stream
	// -----------------------------------
	// Start playing to trigger the OnPacketRTPAny callback function :
	_, err = client.Play(nil)
	if err != nil {
		log.Printf("Error during PLAY: %v\n", err)
	}

	// Run for infinity until explicit exit :
	log.Println("Streaming... Press Ctrl+C to exit.")
	select {}
}
