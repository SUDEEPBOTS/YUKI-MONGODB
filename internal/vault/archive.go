/**
 * =================================================================================================
 *   __     __ _    _  _  __ _____     __  __  ____   _   _   ____   ____   ____   ____  
 *   \ \   / /| |  | || |/ /|_   _|   |  \/  |/ __ \ | \ | | / ___| / __ \ |  _ \ | __ ) 
 *    \ \ / / | |  | || ' /   | |     | |\/| || |  | ||  \| || |  _ | |  | || | | ||  _ \ 
 *     \ V /  | |__| || . \  _| |_    | |  | || |__| || |\  || |_| || |__| || |_| || |_) |
 *      \_/    \____/ |_|\_\|_____|   |_|  |_| \____/ |_| \_| \____| \____/ |____/ |____/  
 *
 * =================================================================================================
 * Project       : YUKI-MONGODB
 * Official Repo : https://github.com/SUDEEPBOTS/YUKI-MONGODB
 * Description   : Database Backup Snapshot & Archiving Compression Engine
 * Maintainer    : SUDEEPBOTS <https://github.com/SUDEEPBOTS>
 *
 * Copyright (c) 2026 SUDEEPBOTS. All rights reserved.
 * Licensed under the MIT License (https://opensource.org/licenses/MIT)
 * =================================================================================================
 */

package vault

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CreateMongoDump executes mongodump into the specified directory
func CreateMongoDump(dumpDir string, port int) error {
	_ = os.RemoveAll(dumpDir)
	cmd := exec.Command("mongodump", "--port", fmt.Sprintf("%d", port), "--out", dumpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mongodump failed: %v, output: %s", err, string(output))
	}
	return nil
}

// RestoreMongoDump executes mongorestore into local instance with --drop
func RestoreMongoDump(dumpDir string, port int) error {
	// Look for nested mongodump or direct directories
	target := dumpDir
	if fi, err := os.Stat(filepath.Join(dumpDir, "mongodump")); err == nil && fi.IsDir() {
		target = filepath.Join(dumpDir, "mongodump")
	}

	cmd := exec.Command("mongorestore", "--port", fmt.Sprintf("%d", port), "--drop", target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mongorestore failed: %v, output: %s", err, string(output))
	}
	log.Printf("[Archive] mongorestore completed: %s", strings.TrimSpace(string(output)))
	return nil
}

// CompressToTarGz compresses a directory into a .tar.gz archive in pure Go
func CompressToTarGz(sourceDir, targetTarGz string) error {
	tarFile, err := os.Create(targetTarGz)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	baseDir := filepath.Base(sourceDir)

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		tarPath := filepath.Join(baseDir, relPath)
		if info.IsDir() {
			tarPath += "/"
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(tarPath)

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}

// ExtractTarGz extracts a .tar.gz archive into the target directory in pure Go
func ExtractTarGz(tarGzPath, targetDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}
