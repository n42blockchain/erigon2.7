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
This command detects v1.1 files by content (not filename) and strips the header.

Note: Erigon 3.x v1.1 files may use "v1-" filename prefix but have different internal format.

Example:
  snapshots downgrade /path/to/snapshots
  snapshots downgrade --dry-run /path/to/snapshots
  snapshots downgrade --types=headers,bodies /path/to/snapshots`,
}

const (
	v11HeaderSize = 32
)

// isV11Format detects if a file is in v1.1 format by checking the header content.
// V1.1 format (Erigon 3.x) has a 32-byte header before the actual data.
// V1.0 format starts directly with wordsCount, emptyWordsCount, dictSize.
// We detect v1.1 by checking if the values at offset 0 are unreasonable for v1.0.
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

	if stat.Size() < 64 { // Need at least 32 (header) + 24 (v1.0 fields) + some data
		return false, nil // File too small
	}

	header := make([]byte, 64)
	if _, err := io.ReadFull(f, header); err != nil {
		return false, err
	}

	// Try parsing as V1.0 format (starts with wordsCount, emptyWordsCount, dictSize)
	wordsCount := binary.BigEndian.Uint64(header[:8])
	emptyWordsCount := binary.BigEndian.Uint64(header[8:16])
	dictSize := binary.BigEndian.Uint64(header[16:24])

	// If these values are unreasonable for V1.0, it's likely V1.1 format
	// V1.1 format: first 32 bytes are header, then wordsCount at offset 32
	if dictSize > uint64(stat.Size()) || dictSize > 1<<40 || wordsCount > 1<<40 {
		// Verify by checking if values at offset 32 make sense
		wordsCountV11 := binary.BigEndian.Uint64(header[32:40])
		emptyWordsCountV11 := binary.BigEndian.Uint64(header[40:48])
		dictSizeV11 := binary.BigEndian.Uint64(header[48:56])

		// Check if V1.1 values are reasonable
		if dictSizeV11 <= uint64(stat.Size()) && wordsCountV11 < 1<<40 && emptyWordsCountV11 <= wordsCountV11 {
			return true, nil
		}
	}

	// Additional check: if emptyWordsCount > wordsCount, it's definitely wrong for V1.0
	if emptyWordsCount > wordsCount && wordsCount > 0 {
		return true, nil
	}

	return false, nil
}

// getV10FileName converts a v1.1 filename to v1.0 filename
// e.g., v1.1-000000-000500-headers.seg -> v1-000000-000500-headers.seg
func getV10FileName(name string) string {
	if strings.HasPrefix(name, "v1.1-") {
		return "v1-" + name[5:]
	}
	return name
}

// convertV11ToV10 converts a v1.1 file to v1.0 format by stripping the 32-byte header
// and optionally renaming the file from v1.1-xxx to v1-xxx
func convertV11ToV10(srcPath string, keepOriginal bool, renameFile bool) (string, error) {
	srcDir := filepath.Dir(srcPath)
	srcName := filepath.Base(srcPath)
	
	// Determine destination filename
	dstName := srcName
	if renameFile {
		dstName = getV10FileName(srcName)
	}
	dstPath := filepath.Join(srcDir, dstName)
	
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	stat, err := srcFile.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat source: %w", err)
	}

	// Skip the 32-byte header
	if _, err := srcFile.Seek(v11HeaderSize, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek: %w", err)
	}

	// Create temporary output file
	tmpPath := dstPath + ".v10.tmp"
	dstFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
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
		return "", fmt.Errorf("failed to copy data: %w", err)
	}

	expectedSize := stat.Size() - v11HeaderSize
	if written != expectedSize {
		return "", fmt.Errorf("size mismatch: expected %d, got %d", expectedSize, written)
	}

	if err := dstFile.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync: %w", err)
	}
	dstFile.Close()
	srcFile.Close()

	// Handle original file
	if keepOriginal {
		// Rename original to .v11.bak
		bakPath := srcPath + ".v11.bak"
		if err := os.Rename(srcPath, bakPath); err != nil {
			os.Remove(tmpPath)
			return "", fmt.Errorf("failed to backup original: %w", err)
		}
	} else {
		// Remove original
		if err := os.Remove(srcPath); err != nil {
			os.Remove(tmpPath)
			return "", fmt.Errorf("failed to remove original: %w", err)
		}
	}

	// Move temp to destination path
	if err := os.Rename(tmpPath, dstPath); err != nil {
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}

	return dstName, nil
}

func downgrade(cliCtx *cli.Context) error {
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

	fmt.Printf("Scanning for v1.1 format snapshot files in: %s (dry-run: %v)\n", snapshotsDir, dryRun)

	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var converted, skipped, alreadyV10 int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if it's a segment file (any version prefix: v1-, v1.1-, etc.)
		if !strings.HasSuffix(name, ".seg") {
			continue
		}

		// Must start with version prefix
		if !strings.HasPrefix(name, "v") {
			continue
		}

		// Parse file info to check type filter
		fileInfo, _, ok := snaptype.ParseFileName(snapshotsDir, name)
		if !ok {
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

		// Check if filename has v1.1 prefix (needs renaming)
		needsRename := strings.HasPrefix(name, "v1.1-")
		
		// Check if file content is v1.1 format (has 32-byte header)
		isV11Content, err := isV11Format(srcPath)
		if err != nil {
			fmt.Printf("  Warning: Failed to check file format %s: %v\n", name, err)
			continue
		}

		// Skip if neither filename nor content indicates v1.1
		if !needsRename && !isV11Content {
			alreadyV10++
			continue
		}

		if dryRun {
			info, _ := entry.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			dstName := name
			if needsRename {
				dstName = getV10FileName(name)
			}
			fmt.Printf("  [DRY-RUN] Would convert: %s -> %s (v1.1_content=%v, size=%.2f MB)\n",
				name, dstName, isV11Content, float64(size)/1024/1024)
			converted++
			continue
		}

		// Convert: strip header if v1.1 content, rename if v1.1 filename
		if isV11Content {
			logger.Info("Converting v1.1 to v1.0", "file", name, "rename", needsRename)
			dstName, err := convertV11ToV10(srcPath, keepOriginal, needsRename)
			if err != nil {
				logger.Error("Failed to convert", "file", name, "error", err)
				continue
			}
			
			// Also handle associated .idx files
			srcIdxPath := strings.TrimSuffix(srcPath, ".seg") + ".idx"
			if _, err := os.Stat(srcIdxPath); err == nil {
				if keepOriginal {
					os.Rename(srcIdxPath, srcIdxPath+".v11.bak")
				} else {
					os.Remove(srcIdxPath)
				}
				logger.Info("Removed old index (needs regeneration)", "file", filepath.Base(srcIdxPath))
			}
			
			logger.Info("Converted", "from", name, "to", dstName)
		} else if needsRename {
			// Only rename, no content conversion needed
			dstName := getV10FileName(name)
			dstPath := filepath.Join(snapshotsDir, dstName)
			
			if keepOriginal {
				// Copy instead of rename
				srcFile, err := os.Open(srcPath)
				if err != nil {
					logger.Error("Failed to open for copy", "file", name, "error", err)
					continue
				}
				dstFile, err := os.Create(dstPath)
				if err != nil {
					srcFile.Close()
					logger.Error("Failed to create destination", "file", dstName, "error", err)
					continue
				}
				_, err = io.Copy(dstFile, srcFile)
				srcFile.Close()
				dstFile.Close()
				if err != nil {
					os.Remove(dstPath)
					logger.Error("Failed to copy", "file", name, "error", err)
					continue
				}
				os.Rename(srcPath, srcPath+".v11.bak")
			} else {
				if err := os.Rename(srcPath, dstPath); err != nil {
					logger.Error("Failed to rename", "file", name, "error", err)
					continue
				}
			}
			
			// Also rename associated .idx files
			srcIdxPath := strings.TrimSuffix(srcPath, ".seg") + ".idx"
			if _, err := os.Stat(srcIdxPath); err == nil {
				dstIdxName := getV10FileName(strings.TrimSuffix(name, ".seg") + ".idx")
				dstIdxPath := filepath.Join(snapshotsDir, dstIdxName)
				if keepOriginal {
					os.Rename(srcIdxPath, srcIdxPath+".v11.bak")
				} else {
					os.Rename(srcIdxPath, dstIdxPath)
				}
			}
			
			logger.Info("Renamed", "from", name, "to", dstName)
		}

		converted++
	}

	logger.Info("Scan complete",
		"v1.1_found", converted,
		"already_v1.0", alreadyV10,
		"skipped_by_filter", skipped,
		"dry_run", dryRun)

	if !dryRun && converted > 0 {
		logger.Info("Conversion complete. Index files may need to be regenerated on next startup.")
	}

	return nil
}
