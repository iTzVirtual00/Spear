package cmd

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"spear/config"
	"spear/handlers/share"
	"spear/utils"
	"spear/utils/share_vfs" // NEW: our virtual fs builder

	"github.com/spf13/cobra"
)

var (
	shareParams config.ShareParams

	// alias=path pairs (repeatable or comma-separated)
	mountFlag map[string]string
)

var shareCmd = &cobra.Command{
	Use:   "share <path-or-alias=path> [<path-or-alias=path>...]",
	Short: "Serve one or more files/folders over HTTP",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Collect alias groups.
		// alias "" means "root" (no alias).
		groups := map[string][]string{}
		// from flags
		for alias, p := range mountFlag {
			groups[alias] = append(groups[alias], p)
		}
		// from positionals
		for _, a := range args {
			if alias, p, ok := splitAliasPair(a); ok {
				groups[alias] = append(groups[alias], p)
			} else {
				groups[""] = append(groups[""], a)
			}
		}

		builder := share_vfs.NewBuilder()

		// TODO: Try our best to avoid collisions (currently unchecked)

		// Build mounts per alias
		for alias, paths := range groups {
			// canonicalize group to absolute real paths
			var canon []*utils.Path
			for _, p := range paths {
				cp, err := utils.NewPath(p).CanonicalPath()
				if err != nil {
					return fmt.Errorf("path %q: %w", p, err)
				}
				canon = append(canon, cp)
			}

			if alias == "" {
				// Unaliased: mount each at root using basename (dirs as subtrees, files as files)
				for _, cp := range canon {
					isDir := cp.IsDir()
					name := cp.Base()
					if isDir {
						builder.MountDir(name, cp, share_vfs.DirMountKeep) // /<basename>/...
					} else {
						builder.MountFile(name, cp) // /<basename>
					}
				}
				continue
			}

			// Aliased group: apply your rules
			var nFiles, nDirs int
			for _, cp := range canon {
				isDir := cp.IsDir()
				if nFiles != 0 || nDirs > 1 {
					// Early out: we already know we have a mixed or multi-dir group
					break
				}
				if isDir {
					nDirs++
				} else {
					nFiles++
				}
			}

			switch {
			case nDirs == 1 && nFiles == 0:
				// Single directory only → lift contents into /alias/
				builder.MountDir(alias, canon[0], share_vfs.DirMountStrip) // /alias/<contents-of-dir>
			default:
				// Any files OR multiple directories → mount basenames under /alias/
				// Create the alias dir as a virtual container
				builder.EnsureDir(alias)
				for _, cp := range canon {
					isDir := cp.IsDir()
					name := cp.Base()
					if isDir {
						// /alias/<basename>/...
						builder.MountDir(filepath.ToSlash(filepath.Join(alias, name)), cp, share_vfs.DirMountKeep)
					} else {
						// /alias/<basename>
						builder.MountFile(filepath.ToSlash(filepath.Join(alias, name)), cp)
					}
				}
			}
		}

		var fsys fs.FS = builder.Build()
		shareParams.MountFS = fsys

		share.SpearShare(&spearConfig, &shareParams)
		return nil
	},
}

func init() {
	shareCmd.Flags().StringVarP(&shareParams.ListenAddress, "listen", "l", "0.0.0.0:9000", "Address to listen on (host:port)")
	shareCmd.Flags().IntVarP(&shareParams.MaxDownloads, "max-downloads", "m", 0, "Maximum number of downloads (0 = unlimited)")
	shareCmd.Flags().IntVarP(&shareParams.ExpiresIn, "expires-in", "e", 0, "Time in seconds until the share expires (0 = no expiration)")
	shareCmd.Flags().StringArrayVarP(&shareParams.AllowContacts, "allow", "a", []string{}, "List of contacts allowed to access the share")
	shareCmd.Flags().BoolVarP(&shareParams.NatPunch, "nat-punch", "n", false, "Enable NAT punchthrough")

	shareCmd.Flags().StringToStringVarP(
		&mountFlag,
		"mount", "M",
		nil,
		"Mount alias=path (repeatable or comma-separated), e.g. -M pics=~/Pictures,docs=./Docs",
	)

	rootCmd.AddCommand(shareCmd)
}

// --- helpers ---

func splitAliasPair(s string) (alias, path string, ok bool) {
	// split first '=' ONLY
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", s, false
}
