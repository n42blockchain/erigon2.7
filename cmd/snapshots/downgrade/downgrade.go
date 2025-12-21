package downgrade

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/erigontech/erigon-lib/downloader/snaptype"
	"github.com/erigontech/erigon/cmd/snapshots/flags"
	"github.com/erigontech/erigon/cmd/snapshots/sync"
	"github.com/erigontech/erigon/cmd/utils"
	"github.com/erigontech/erigon/turbo/logging"
)

var (
	DryRunFlag = cli.BoolFlag{
		Name:     "dry-run",
		Usage:    `Only show what would be converted without actually doing it`,
		Required: false,
	}

	KeepOriginalFlag = cli.BoolFlag{
		Name:     "keep-original",
		Usage:    `Keep original v1.1 files with .v11.bak suffix`,
		Required: false,
		Value:    true,
	}
)

var Command = cli.Command{
	Action:    downgrade,
	Name:      "downgrade",
	Usage:     "downgrade v1.1 snapshot segments to v1.0 format",
	ArgsUsage: "<snapshots-dir>",
	Flags: []cli.Flag{
		&flags.SegTypes,
		&DryRunFlag,
		&KeepOriginalFlag,
		&utils.DataDirFlag,
		&logging.LogVerbosityFlag,
		&logging.LogConsoleVerbosityFlag,
		&logging.LogDirVerbosityFlag,
	},
	Description: `Converts v1.1 format snapshot files (Erigon 3.x) to v1.0 format (Erigon 2.x).
The v1.1 format has a 32-byte header that is not present in v1.0 format.
This command strips the header from v1.1 files and renames them to v1.0.

Example:
  snapshots downgrade /path/to/snapshots
  snapshots downgrade --dry-run /path/to/snapshots
  snapshots downgrade --types=headers,bodies /path/to/snapshots`,
}

const (
	v11HeaderSize = 32
)

// isV11Format detects if a file is in v1.1 format by checking the header
func isV11Format(filePath string) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return false, err
	}

	if stat.Size() < 32 {
		return false, nil // File too small
	}

	header := make([]byte, 32)
	if _, err := io.ReadFull(f, header); err != nil {
		return false, err
	}

	// V1.0 format starts with wordsCount (8 bytes), emptyWordsCount (8 bytes), dictSize (8 bytes)
	// V1.1 format has a 32-byte header, then these fields

	// Read as V1.0 format
	wordsCount := binary.BigEndian.Uint64(header[:8])
	dictSize := binary.BigEndian.Uint64(header[16:24])

	// If dictSize is unreasonably large (> file size), it's likely V1.1 format
	if dictSize > uint64(stat.Size()) || dictSize > 1<<40 || wordsCount > 1<<40 {
		return true, nil
	}

	return false, nil
}

// convertV11ToV10 converts a v1.1 file to v1.0 format by stripping the 32-byte header
func convertV11ToV10(srcPath, dstPath string, keepOriginal bool) error {
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	stat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	// Skip the 32-byte header
	if _, err := srcFile.Seek(v11HeaderSize, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// Create temporary output file
	tmpPath := dstPath + ".tmp"
	dstFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer func() {
		dstFile.Close()
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	// Copy the rest of the file
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	expectedSize := stat.Size() - v11HeaderSize
	if written != expectedSize {
		return fmt.Errorf("size mismatch: expected %d, got %d", expectedSize, written)
	}

	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}
	dstFile.Close()

	// Handle original file
	if keepOriginal {
		// Rename original to .v11.bak
		bakPath := srcPath + ".v11.bak"
		if err := os.Rename(srcPath, bakPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to backup original: %w", err)
		}
	} else {
		// Remove original
		if err := os.Remove(srcPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to remove original: %w", err)
		}
	}

	// Move temp to final destination
	if err := os.Rename(tmpPath, dstPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// getV10FileName converts a v1.1 filename to v1.0 filename
// e.g., v1.1-000000-000500-headers.seg -> v1-000000-000500-headers.seg
func getV10FileName(name string) string {
	// Replace v1.1- prefix with v1-
	if strings.HasPrefix(name, "v1.1-") {
		return "v1-" + name[5:]
	}
	return name
}

func downgrade(cliCtx *cli.Context) error {
	logger := sync.Logger(cliCtx.Context)

	var snapshotsDir string

	if cliCtx.Args().Len() > 0 {
		snapshotsDir = cliCtx.Args().Get(0)
	} else if dataDir := cliCtx.String(utils.DataDirFlag.Name); dataDir != "" {
		snapshotsDir = filepath.Join(dataDir, "snapshots")
	} else {
		return fmt.Errorf("please provide snapshots directory as argument or use --datadir flag")
	}

	dryRun := cliCtx.Bool(DryRunFlag.Name)
	keepOriginal := cliCtx.Bool(KeepOriginalFlag.Name)

	// Parse segment types filter
	typeValues := cliCtx.StringSlice(flags.SegTypes.Name)
	snapTypes := make(map[string]bool)
	for _, val := range typeValues {
		snapTypes[val] = true
	}

	logger.Info("Scanning for v1.1 snapshot files", "dir", snapshotsDir, "dryRun", dryRun)

	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var converted, skipped int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if it's a v1.1 segment file
		if !strings.HasPrefix(name, "v1.1-") || !strings.HasSuffix(name, ".seg") {
			continue
		}

		// Parse file info to check type filter
		fileInfo, _, ok := snaptype.ParseFileName(snapshotsDir, name)
		if !ok {
			logger.Warn("Failed to parse filename", "name", name)
			continue
		}

		// Apply type filter if specified
		if len(snapTypes) > 0 && fileInfo.Type != nil {
			if !snapTypes[fileInfo.Type.Name()] {
				skipped++
				continue
			}
		}

		srcPath := filepath.Join(snapshotsDir, name)
		dstName := getV10FileName(name)
		dstPath := filepath.Join(snapshotsDir, dstName)

		// Verify it's actually v1.1 format
		isV11, err := isV11Format(srcPath)
		if err != nil {
			logger.Warn("Failed to check file format", "file", name, "error", err)
			continue
		}

		if !isV11 {
			logger.Debug("File is already v1.0 format, skipping", "file", name)
			skipped++
			continue
		}

		if dryRun {
			logger.Info("Would convert", "from", name, "to", dstName)
			converted++
			continue
		}

		logger.Info("Converting", "from", name, "to", dstName)

		if err := convertV11ToV10(srcPath, dstPath, keepOriginal); err != nil {
			logger.Error("Failed to convert", "file", name, "error", err)
			continue
		}

		// Also handle associated files (.idx, .torrent)
		for _, ext := range []string{".idx", ".torrent"} {
			srcAssoc := strings.TrimSuffix(srcPath, ".seg") + ext
			if _, err := os.Stat(srcAssoc); err == nil {
				if keepOriginal {
					// Backup associated files
					os.Rename(srcAssoc, srcAssoc+".v11.bak")
				} else {
					// Remove old associated files, they need regeneration
					os.Remove(srcAssoc)
				}
				// Note: idx files may need regeneration, torrent files definitely need regeneration
				logger.Info("Note: Please regenerate index/torrent for", "file", dstName)
			}
		}

		converted++
	}

	if dryRun {
		logger.Info("Dry run complete", "wouldConvert", converted, "skipped", skipped)
	} else {
		logger.Info("Conversion complete", "converted", converted, "skipped", skipped)
		if converted > 0 {
			logger.Info("Note: Index files (.idx) may need to be regenerated")
		}
	}

	return nil
}

