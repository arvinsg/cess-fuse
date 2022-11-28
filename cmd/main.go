package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arvinsg/cess-fuse/pkg/fs"
	"github.com/urfave/cli"
)

func main() {
	var (
		app   = NewApp()
		flags *fs.Flags
	)

	app.Action = func(c *cli.Context) (err error) {
		// Populate and parse flags.
		flags = PopulateFlags(c)
		if flags == nil {
			cli.ShowAppHelp(c)
			return
		}

		defer func() {
			time.Sleep(time.Second)
			flags.Cleanup()
		}()
		cloud, err := store.NewCessStorage(ParseCESSConfig(flags))
		if err != nil {
			fmt.Fprintf(os.Stderr, "create cess storage fail, err: %v\n", err)
			return
		}

		fs, mfs, err := fs.MountFS(context.Background(), cloud, flags)
		registerSigINTHandler(fs, flags.MountPoint)
		fmt.Fprintln(os.Stdout, "File system has been successfully mounted.")

		// Wait for the file system to be unmounted.
		err = mfs.Join(context.Background())
		if err != nil {
			return fmt.Errorf("MountedFileSystem.Join: %v", err)
		}

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to mount file system, err:%v \n", err)
		os.Exit(1)
	}
}

func registerSigINTHandler(fs *fs.FileSystem, mountPoint string) {
	// Register for SIGINT.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)

	// Start a goroutine that will unmount when the signal is received.
	go func() {
		for {
			s := <-signalChan
			if s == syscall.SIGUSR1 {
				fs.SigUsr1()
				continue
			}

			err := fs.TryUnmountFS(mountPoint)
			if err != nil {
				fmt.Fprintf(os.Stderr, "try unmount fail, err:%v \n", err)
			} else {
				return
			}
		}
	}()
}
