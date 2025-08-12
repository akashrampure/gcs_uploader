package main

import "gcsuploader/utils"

func main() {
	utils.Start("8080", "config/credentials.json", "dmtfota")
}
