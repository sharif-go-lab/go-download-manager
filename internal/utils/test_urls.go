package utils

// TestURLs contains a list of reliable file URLs for testing downloads.
var TestURLs = map[string]string{
	"1MB":   "https://speed.hetzner.de/1MB.bin",
	"5MB":   "https://speed.hetzner.de/5MB.bin",
	"10MB":  "https://download.thinkbroadband.com/10MB.zip",
	"50MB":  "https://speed.hetzner.de/50MB.bin",
	"100MB": "https://download.thinkbroadband.com/100MB.zip",
	"500MB": "https://speed.hetzner.de/500MB.bin",
	"1GB":   "https://speed.hetzner.de/1GB.bin",
	"10MB_OVH": "http://proof.ovh.net/files/10Mb.dat", // use this (others don't work)
	"100MB_OVH": "https://proof.ovh.net/files/100Mb.dat", // and this
} 