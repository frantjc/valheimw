package valheim

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	xslice "github.com/frantjc/x/slice"
)

var (
	AdminListName     = "adminlist.txt"
	BannedListName    = "bannedlist.txt"
	PermittedListName = "permittedlist.txt"
)

type PlayerLists struct {
	AdminIDs     []int64
	BannedIDs    []int64
	PermittedIDs []int64
}

func ReadPlayerList(r io.Reader) ([]int64, error) {
	var (
		scanner   = bufio.NewScanner(r)
		playerIDs = []int64{}
	)

	for scanner.Scan() {
		line := strings.TrimSpace(
			strings.Split(
				strings.Split(
					scanner.Text(),
					"#",
				)[0],
				"//",
			)[0],
		)

		if line == "" {
			continue
		}

		if playerID, err := strconv.Atoi(line); err == nil {
			playerIDs = append(playerIDs, int64(playerID))
		}
	}

	return playerIDs, scanner.Err()
}

func WritePlayerList(w io.Writer, playerIDs []int64) error {
	for _, playerID := range playerIDs {
		if _, err := fmt.Fprintln(w, playerID); err != nil {
			return err
		}
	}

	return nil
}

func WritePlayerLists(savedir string, playerLists *PlayerLists) error {
	if err := WritePlayerListFile(filepath.Join(savedir, AdminListName), playerLists.AdminIDs); err != nil {
		return err
	}

	if err := WritePlayerListFile(filepath.Join(savedir, BannedListName), playerLists.BannedIDs); err != nil {
		return err
	}

	return WritePlayerListFile(filepath.Join(savedir, PermittedListName), playerLists.PermittedIDs)
}

func WritePlayerListFile(name string, playerIDs []int64) error {
	if len(playerIDs) > 0 {
		f, err := os.Create(name)
		if err != nil {
			return err
		}

		currentPlayerIDs, err := ReadPlayerList(f)
		if err != nil {
			return err
		}

		playerIDs = xslice.Unique(append(currentPlayerIDs, playerIDs...))

		return WritePlayerList(f, playerIDs)
	}

	return nil
}
