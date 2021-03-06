// This file is part of go-autoruns.
// Copyright (c) 2018-2021 Claudio Guarnieri
// See the file 'LICENSE' for copying permission.

package autoruns

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/botherder/go-savetime/hashes"
)

const (
	LINUX_ENTRY_SYSTEMD = "systemd"
)

// This function just invokes all the platform-dependant functions.
func GetAllAutoruns() (records []*Autorun) {
	records = append(records, LinuxGetSystemd()...)
	return
}

func systemdToAutorun(servicePath string) (*Autorun, error) {
	reader, err := os.Open(servicePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	autorun := Autorun{
		Location: servicePath,
		Type:     LINUX_ENTRY_SYSTEMD,
	}

	regexSection := regexp.MustCompile("\\[.*\\]")
	inSection := ""
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if regexSection.MatchString(line) {
			inSection = line
		}

		switch inSection {
		case "[Service]":
			if strings.HasPrefix(line, "ExecStart=") {
				parseShellInvocation(line, &autorun)
			}
		case "[D-BUS Service]":
			if strings.HasPrefix(line, "Exec=") {
				parseShellInvocation(line, &autorun)
			}
		}
	}

	autorun.MD5, _ = hashes.FileMD5(autorun.ImagePath)
	autorun.SHA1, _ = hashes.FileSHA1(autorun.ImagePath)
	autorun.SHA256, _ = hashes.FileSHA256(autorun.ImagePath)

	return &autorun, nil
}

func LinuxGetSystemd() (records []*Autorun) {
	folders := []string{
		"/etc/systemd/system/",
		"/usr/share/dbus-1/system-services/",
	}

	for _, folder := range folders {
		// Check if the folders exists.
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			continue
		}

		// Get list of files in folder.
		filesList, err := ioutil.ReadDir(folder)
		if err != nil {
			continue
		}

		// Loop through all files in folder.
		for _, fileEntry := range filesList {
			// Skip all files that don't end with .service.
			if !(strings.HasSuffix(fileEntry.Name(), ".service")) {
				continue
			}

			filePath := filepath.Join(folder, fileEntry.Name())
			record, err := systemdToAutorun(filePath)
			if err != nil {
				continue
			}

			records = append(records, record)
		}
	}

	return
}
