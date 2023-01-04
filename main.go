package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/radekg/boos/cmd/all"
	"github.com/radekg/boos/cmd/backend"
	"github.com/radekg/boos/cmd/frontend"
	"github.com/spf13/cobra"

	crypto_rand "crypto/rand"
	math_rand "math/rand"
)

var rootCmd = &cobra.Command{
	Use:   "boos",
	Short: "boos",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(1)
	},
}

func seedRandom() {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		panic("cannot seed math/rand package with cryptographically secure random number generator")
	}
	math_rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
}

func init() {
	seedRandom()
	rootCmd.AddCommand(all.Command)
	rootCmd.AddCommand(backend.Command)
	rootCmd.AddCommand(frontend.Command)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
