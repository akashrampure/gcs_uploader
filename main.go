package main

import "gcsuploader/utils"

func main() {
	utils.StartServer("8080", "config/credentials.json", "dmtfota")
}
