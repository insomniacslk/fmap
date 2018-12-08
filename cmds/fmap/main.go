package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/insomniacslk/fmap/pkg/fmap"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [file]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	infile := "-"
	if len(flag.Args()) > 0 {
		infile = flag.Arg(0)
	}
	var (
		fd  *os.File
		err error
	)
	if infile == "-" {
		log.Print("Reading from stdin")
		fd = os.Stdin
	} else {
		fd, err = os.Open(infile)
		if err != nil {
			log.Fatal(err)
		}
		defer fd.Close()
	}
	flash, err := fmap.Parse(fd)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("===================== BEFORE =====================")
	fmt.Printf("%+v\n", flash)
	fmt.Println(flash.ToFlashmap())

	biosSec := flash.Find("SI_BIOS", false)
	if biosSec != nil {
		log.Print("SI_BIOS section found.")
	} else {
		log.Fatal("No SI_BIOS section found")
	}

	// after removing RW_SECTION_B, the COREBOOT section will be increased by
	// this size
	freeSpaceSize := biosSec.Size
	if biosSec.Remove("RW_SECTION_B", false) {
		log.Print("Removed RW_SECTION_B.")
	} else {
		log.Fatal("Could not find and remove RW_SECTION_B")
	}

	log.Print("Compacting BIOS sub-sections")
	if biosSec.Defrag() {
		log.Print("Successfully defragmented BIOS section")
	}

	log.Printf("Expanding WP_RO->RO_SECTION->COREBOOT by 0x%x", freeSpaceSize)
	wpRO := biosSec.Sections[len(biosSec.Sections)-1]
	if wpRO.Name != "WP_RO" {
		log.Fatalf("Name is not WP_RO: got %s", wpRO.Name)
	}
	wpRO.Size += freeSpaceSize
	roSection := wpRO.Sections[len(wpRO.Sections)-1]
	if roSection.Name != "RO_SECTION" {
		log.Fatalf("Name is not RO_SECTION: got %s", roSection.Name)
	}
	roSection.Size += freeSpaceSize
	payload := roSection.Sections[len(roSection.Sections)-1]
	if payload.Name != "COREBOOT" {
		log.Fatalf("Name is not COREBOOT: got %s", payload.Name)
	}
	payload.Size += freeSpaceSize

	log.Print("===================== AFTER =====================")
	fmt.Println(flash.ToFlashmap())
}
