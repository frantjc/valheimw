package valheim

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/frantjc/go-fn"
)

var (
	AdminListName     = "adminlist.txt"
	BannedListName    = "bannedlist.txt"
	PermittedListName = "permittedlist.txt"
)

type PlayerLists struct {
	Admins    []int
	Banned    []int
	Permitted []int
}

func ReadPlayerList(r io.Reader) ([]int, error) {
	var (
		scanner = bufio.NewScanner(r)
		players = []int{}
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

		if player, err := strconv.Atoi(line); err == nil {
			players = append(players, player)
		}
	}

	return players, scanner.Err()
}

func WritePlayerList(w io.Writer, players []int) error {
	for _, player := range players {
		if _, err := fmt.Fprintln(w, player); err != nil {
			return err
		}
	}

	return nil
}

func WritePlayerLists(savedir string, playerLists *PlayerLists) error {
	if err := WritePlayerListFile(filepath.Join(savedir, AdminListName), playerLists.Admins); err != nil {
		return err
	}

	if err := WritePlayerListFile(filepath.Join(savedir, BannedListName), playerLists.Banned); err != nil {
		return err
	}

	return WritePlayerListFile(filepath.Join(savedir, PermittedListName), playerLists.Permitted)
}

func WritePlayerListFile(name string, players []int) error {
	if len(players) > 0 {
		f, err := os.Create(name)
		if err != nil {
			return err
		}

		currentPlayers, err := ReadPlayerList(f)
		if err != nil {
			return err
		}

		players = fn.Filter(players, func(player int, _ int) bool {
			return !fn.Includes(currentPlayers, player)
		})

		return WritePlayerList(f, players)
	}

	return nil
}
