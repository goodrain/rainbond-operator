package commonutil

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	commentChar string = "#"
	//StartOfSection writ hosts start
	StartOfSection = "# Mirror repository and IP mapping"
	//EndOfSection writ hosts end
	EndOfSection = "# End of Section"
	eol          = "\n"
)

// FileExists checks if a file exists and is not a directory.
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// IsFile checks if a file is a file.
func IsFile(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// Untar reads the gzip-compressed tar file from r and writes it into dir.
func Untar(r io.Reader, dir string) error {
	return untar(r, dir)
}

func untar(r io.Reader, dir string) (err error) {
	t0 := time.Now()
	nFiles := 0
	madeDir := map[string]bool{}
	defer func() {
		td := time.Since(t0)
		if err == nil {
			log.Printf("extracted tarball into %s: %d files, %d dirs (%v)", dir, nFiles, len(madeDir), td)
		} else {
			log.Printf("error extracting tarball into %s after %d files, %d dirs, %v: %v", dir, nFiles, len(madeDir), td, err)
		}
	}()
	zr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("requires gzip-compressed body: %v", err)
	}
	tr := tar.NewReader(zr)
	loggedChtimesError := false
	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("tar reading error: %v", err)
			return fmt.Errorf("tar error: %v", err)
		}
		if !validRelPath(f.Name) {
			return fmt.Errorf("tar contained invalid name error %q", f.Name)
		}
		rel := filepath.FromSlash(f.Name)
		abs := filepath.Join(dir, rel)

		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsRegular():
			// Make the directory. This is redundant because it should
			// already be made by a directory entry in the tar
			// beforehand. Thus, don't check for errors; the next
			// write will fail with the same error.
			dir := filepath.Dir(abs)
			if !madeDir[dir] {
				if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
					return err
				}
				madeDir[dir] = true
			}
			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return err
			}
			n, err := io.Copy(wf, tr)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return fmt.Errorf("error writing to %s: %v", abs, err)
			}
			if n != f.Size {
				return fmt.Errorf("only wrote %d bytes to %s; expected %d", n, abs, f.Size)
			}
			modTime := f.ModTime
			if modTime.After(t0) {
				// Clamp modtimes at system time. See
				// golang.org/issue/19062 when clock on
				// buildlet was behind the gitmirror server
				// doing the git-archive.
				modTime = t0
			}
			if !modTime.IsZero() {
				if err := os.Chtimes(abs, modTime, modTime); err != nil && !loggedChtimesError {
					// benign error. Gerrit doesn't even set the
					// modtime in these, and we don't end up relying
					// on it anywhere (the gomote push command relies
					// on digests only), so this is a little pointless
					// for now.
					log.Printf("error changing modtime: %v (further Chtimes errors suppressed)", err)
					loggedChtimesError = true // once is enough
				}
			}
			nFiles++
		case mode.IsDir():
			if err := os.MkdirAll(abs, 0755); err != nil {
				return err
			}
			madeDir[abs] = true
		default:
			return fmt.Errorf("tar file entry %s contained unsupported file type %v", f.Name, mode)
		}
	}
	return nil
}

func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}

// WriteHosts set rainbond imagehub and ip to local host file
func WriteHosts(hostspath, ip string) {
	// open hostfile in operator
	hostFile, err := os.OpenFile(hostspath, os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		logrus.Error("open host file error", err)
		return
		panic(err)
	}
	defer hostFile.Close()
	//  check ip and rainbond hub url is exist
	r := bufio.NewReader(hostFile)
	for {
		line, err := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if err != nil && err != io.EOF {
			logrus.Error("read line to host file error", err)
			panic(err)
		}
		if err == io.EOF {
			logrus.Error("read line to host file EOF", err)
			break
		}
		if line == StartOfSection {
			return
		}
	}
	// add rainbond hub url if not exist
	lines := []string{
		eol + StartOfSection + eol,
		ip + " " + "goodrain.me" + eol,
		ip + " " + "region.goodrain.me" + eol,
		EndOfSection + eol,
	}
	writer := bufio.NewWriter(hostFile)
	for _, line := range lines {
		if _, err := writer.WriteString(line); err != nil {
			logrus.Error("write line to host file error", err)
		}
	}
	writer.Flush()
}
