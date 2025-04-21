package main

import (
	"client/torrent"
	"client/torrentfile"
	"crypto/rand"
	"log"
	"os"
)

func createPeerId() *[20]byte {
	var b [20]byte
	_, err := rand.Read(b[:])
	if err != nil {
		log.Fatal("Failed to generate random bytes:", err)
	}
	return &b
}

func writeToFile(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data) // convert [20]byte to []byte
	return err
}

func main() {
	tf, err := torrentfile.Open("../exampletorrents/rdr.torrent")
	if err != nil {
		log.Fatalf("opening torrent - %v", err)
	}
	log.Printf("infohash - %x", tf.InfoHash)
	peerId := createPeerId()
	port := uint16(6881)
	t, err := torrent.New(&tf, peerId, port)
	if err != nil {
		log.Fatalf("create torrent - %v", err)
	}
	log.Println("PEERS FOUND: ", t.Peers)
	buf := t.Download()
	err = writeToFile("../downloaded.bin", buf)
	if err != nil {
		log.Fatalf("writing to file - %v", err)
	}
	log.Printf("END")
}

// package main

// import (
// 	"client/peering"
// 	"client/torrentfile"
// 	"client/tracker"
// 	"encoding/hex"
// 	"fmt"
// 	"log"
// 	"os"
// 	"strings"
// )

// /*
// // func showMessage(window fyne.Window, message string) {
// // 	dialog := widget.NewLabel(message)
// // 	window.SetContent(container.NewVBox(
// // 		dialog,
// // 		widget.NewButton("Close", func() {
// // 			window.Close()
// // 		}),
// // 	))
// // 	window.Show()
// // }

// // // func main() {
// // // 	// Create the application
// // // 	myApp := app.New()
// // // 	myWindow := myApp.NewWindow("BitTorrent App")

// // // 	// Create a toolbar with basic actions
// // // 	toolbar := widget.NewToolbar(
// // // 		widget.NewToolbarAction(theme.FileIcon(), func() {
// // // 			// Add torrent action
// // // 			dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
// // // 				if err != nil {
// // // 					showMessage(myWindow, "Error: "+err.Error())
// // // 					return
// // // 				}
// // // 				if reader == nil {
// // // 					showMessage(myWindow, "No file selected")
// // // 					return
// // // 				}
// // // 				defer reader.Close()

// // // 				// Read the file content
// // // 				data, err := os.ReadFile(reader.URI().Path())
// // // 				if err != nil {
// // // 					showMessage(myWindow, "Failed to read file: "+err.Error())
// // // 					return
// // // 				}
// // // 				showMessage(myWindow, "File content: "+string(data))
// // // 			}, myWindow).Show()
// // // 		}),
// // // 		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
// // // 			// Start torrent action
// // // 			showMessage(myWindow, "Start Torrent")
// // // 		}),
// // // 		widget.NewToolbarAction(theme.MediaPauseIcon(), func() {
// // // 			// Stop torrent action
// // // 			showMessage(myWindow, "Stop Torrent")
// // // 		}),
// // // 	)

// // // 	// Create a list for torrents
// // // 	torrentList := widget.NewList(
// // // 		func() int {
// // // 			return 10 // Dummy list length
// // // 		},
// // // 		func() fyne.CanvasObject {
// // // 			return widget.NewLabel("Torrent Item")
// // // 		},
// // // 		func(id widget.ListItemID, obj fyne.CanvasObject) {
// // // 			obj.(*widget.Label).SetText("Torrent " + string(id+'0'))
// // // 		},
// // // 	)

// // // 	// Details panel
// // // 	detailPanel := widget.NewMultiLineEntry()
// // // 	detailPanel.SetText("Select torrent to see details")

// // // 	// Layout content
// // // 	mainContent := container.NewHSplit(
// // // 		torrentList,
// // // 		container.NewVBox(
// // // 			widget.NewLabel("Details"),
// // // 			detailPanel,
// // // 		),
// // // 	)
// // // 	mainContent.SetOffset(0.3)

// // // 	// Set up the main window content
// // // 	myWindow.SetContent(container.NewBorder(toolbar, nil, nil, nil, mainContent))
// // // 	myWindow.Resize(fyne.NewSize(800, 600))
// // // 	myWindow.ShowAndRun()
// // // }
// */
// func main() {
// 	file, err := os.Open("../exampletorrents/rdr.torrent")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer file.Close()

// 	announces, infoHash, err := torrentfile.Open(file).toTorrentFile()
// 	if err != nil {
// 		fmt.Println("Error decoding torrent file:", err)
// 	}

// 	fmt.Println("Info Hash: ", hex.EncodeToString(infoHash[:]))

// 	fmt.Println("Announcing trackers:", announces)

// 	for _, url := range announces {
// 		strURL := url[0]

// 		fmt.Println("URL: ", strURL)
// 		if strings.HasPrefix(strURL, "http:") {
// 			peers, err := tracker.SendAnnounceHTTP(strURL, string(infoHash[:]))

// 			if err != nil {
// 				fmt.Println("Error GETting ", strURL, ": ", err)
// 			} else {
// 				fmt.Println("Peers from: ", strURL, ": ", peers)
// 			}
// 		} else if strings.HasPrefix(strURL, "udp:") {
// 			endOfURL := strings.LastIndex(strURL, "/")
// 			var udpAddr string
// 			if endOfURL > 6 {
// 				udpAddr = strURL[6:endOfURL]
// 			} else {
// 				udpAddr = strURL[6:]
// 			}
// 			peers, err := tracker.SendUDPRequest(udpAddr, infoHash)

// 			if err != nil {
// 				fmt.Println("UDP Error from", udpAddr, ": ", err)
// 			} else {
// 				fmt.Println("Peers from", strURL, ": ", peers)
// 				for _, peer := range peers {
// 					go func() {
// 						fmt.Println("handshaking peer: ", peer)
// 						peering.PeerHandshake(peer, infoHash)
// 					}()
// 				}
// 			}
// 		}
// 	}
// }
