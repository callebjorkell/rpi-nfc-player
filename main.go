package main

import "github.com/callebjorkell/nfc-player/nfc"

func main() {
	//s, err := sonos.New("Guest Room")
	//if err != nil {
	//	log.Fatal(err)
	//}

	//log.Println(s.Name(), "found")
	//s.PlayDeezer("tr%3A63534071")

	//f, err := os.Create("penis.png")
	//if err != nil {
	//	panic(err)
	//}
	//defer f.Close()

	//label.CreateLabel("14290022", f)

	// ui.Interact()
	nfc.ReadCard()
}
