package main

import (
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/arvinsg/cess-fuse/pkg/fs"
	"github.com/arvinsg/cess-fuse/pkg/storage"
	"github.com/arvinsg/cess-fuse/pkg/utils"
	"github.com/urfave/cli"
)

var flagCategories map[string]string

// Set up custom help text; in particular the usage section.
func filterCategory(flags []cli.Flag, category string) (ret []cli.Flag) {
	for _, f := range flags {
		if flagCategories[f.GetName()] == category {
			ret = append(ret, f)
		}
	}
	return
}

func init() {
	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.Name}} {{if .Flags}}[global options]{{end}} mountpoint
   {{if .Version}}
VERSION:
   {{.Version}}
   {{end}}{{if len .Authors}}
AUTHOR(S):
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
   {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
   {{end}}{{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range category .Flags ""}}{{.}}
   {{end}}
TUNING OPTIONS:
   {{range category .Flags "tuning"}}{{.}}
   {{end}}
MISC OPTIONS:
   {{range category .Flags "misc"}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}
`
}

func NewApp() (app *cli.App) {
	uid, gid := utils.MyUserAndGroup()

	app = &cli.App{
		Name:     "CESS-Fuse",
		Usage:    "Mount cloud storage locally",
		HideHelp: true,
		Writer:   os.Stderr,
		Flags: []cli.Flag{

			cli.BoolFlag{
				Name:  "help, h",
				Usage: "Print this help text and exit successfully.",
			},

			/////////////////////////
			// File system
			/////////////////////////

			cli.StringSliceFlag{
				Name:  "o",
				Usage: "Additional system-specific mount options. Be careful!",
			},

			cli.IntFlag{
				Name:  "dir-mode",
				Value: 0755,
				Usage: "Permission bits for directories. (default: 0755)",
			},

			cli.IntFlag{
				Name:  "file-mode",
				Value: 0644,
				Usage: "Permission bits for files. (default: 0644)",
			},

			cli.IntFlag{
				Name:  "uid",
				Value: uid,
				Usage: "UID owner of all inodes.",
			},

			cli.IntFlag{
				Name:  "gid",
				Value: gid,
				Usage: "GID owner of all inodes.",
			},

			/////////////////////////
			// CESS Storage config
			/////////////////////////

			cli.BoolFlag{
				Name:  "use-content-type",
				Usage: "Set Content-Type according to file extension and /etc/mime.types (default: off)",
			},

			/////////////////////////
			// Tuning
			/////////////////////////

			cli.BoolFlag{
				Name:  "no-implicit-dir",
				Usage: "Assume all directory objects (\"dir/\") exist (default: off)",
			},

			cli.DurationFlag{
				Name:  "stat-cache-ttl",
				Value: time.Minute,
				Usage: "How long to cache StatObject results and inode attributes.",
			},

			cli.DurationFlag{
				Name:  "type-cache-ttl",
				Value: time.Minute,
				Usage: "How long to cache name -> file/dir mappings in directory " +
					"inodes.",
			},

			cli.DurationFlag{
				Name:  "http-timeout",
				Value: 30 * time.Second,
				Usage: "Set the timeout on HTTP requests to S3",
			},

			/////////////////////////
			// Debugging
			/////////////////////////

			cli.BoolFlag{
				Name:  "debug_fuse",
				Usage: "Enable fuse-related debugging output.",
			},
		},
	}

	var funcMap = template.FuncMap{
		"category": filterCategory,
		"join":     strings.Join,
	}

	flagCategories = map[string]string{}

	for _, f := range []string{"region"} {
		flagCategories[f] = "CESS"
	}

	for _, f := range []string{"no-implicit-dir", "stat-cache-ttl", "type-cache-ttl", "http-timeout"} {
		flagCategories[f] = "tuning"
	}

	for _, f := range []string{"help, h", "debug_fuse", "version, v"} {
		flagCategories[f] = "misc"
	}

	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		w = tabwriter.NewWriter(w, 1, 8, 2, ' ', 0)
		var tmplGet = template.Must(template.New("help").Funcs(funcMap).Parse(templ))
		tmplGet.Execute(w, app)
	}

	return
}

func parseOptions(m map[string]string, s string) {
	// NOTE(jacobsa): The man pages don't define how escaping works, and as far
	// as I can tell there is no way to properly escape or quote a comma in the
	// options list for an fstab entry. So put our fingers in our ears and hope
	// that nobody needs a comma.
	for _, p := range strings.Split(s, ",") {
		var name string
		var value string

		// Split on the first equals sign.
		if equalsIndex := strings.IndexByte(p, '='); equalsIndex != -1 {
			name = p[:equalsIndex]
			value = p[equalsIndex+1:]
		} else {
			name = p
		}

		m[name] = value
	}

	return
}

// PopulateFlags adds the flags accepted by run to the supplied flag set, returning the
// variables into which the flags will parse.
func PopulateFlags(c *cli.Context) *fs.Flags {
	flags := &fs.Flags{
		// File system
		MountOptions: make(map[string]string),
		DirMode:      os.FileMode(c.Int("dir-mode")),
		FileMode:     os.FileMode(c.Int("file-mode")),
		Uid:          uint32(c.Int("uid")),
		Gid:          uint32(c.Int("gid")),

		// Tuning,
		ExplicitDir:  c.Bool("no-implicit-dir"),
		StatCacheTTL: c.Duration("stat-cache-ttl"),
		TypeCacheTTL: c.Duration("type-cache-ttl"),
		HTTPTimeout:  c.Duration("http-timeout"),

		// Common Backend Flags
		UseContentType: c.Bool("use-content-type"),

		// Debugging,
		DebugFuse: c.Bool("debug_fuse"),
	}

	// Handle the repeated "-o" flag.
	for _, o := range c.StringSlice("o") {
		parseOptions(flags.MountOptions, o)
	}

	flags.MountPointArg = c.Args()[1]
	flags.MountPoint = flags.MountPointArg

	return flags
}

// TODO
func ParseCESSConfig(c *fs.Flags) *storage.CessConfig {
	return nil
}
