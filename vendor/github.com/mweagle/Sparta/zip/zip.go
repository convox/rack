package zip

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
)

// AddToZip creates a source object (either a file, or a directory that will be recursively
// added) to a previously opened zip.Writer.  The archive path of `source` is relative to the
// `rootSource` parameter.
func AddToZip(zipWriter *zip.Writer, source string, rootSource string, logger *logrus.Logger) error {

	linuxZipName := func(platformValue string) string {
		return strings.Replace(platformValue, "\\", "/", -1)
	}

	fullPathSource, err := filepath.Abs(source)
	if nil != err {
		return err
	}

	appendFile := func(info os.FileInfo) error {
		zipEntryName := source
		if "" != rootSource {
			zipEntryName = fmt.Sprintf("%s/%s", linuxZipName(rootSource), info.Name())
		}
		// Create a header for this zipFile, basically let's see
		// if we can get the executable bits to travel along..
		fileHeader, fileHeaderErr := zip.FileInfoHeader(info)
		if fileHeaderErr != nil {
			return fileHeaderErr
		}
		// Update the name to the proper thing...
		fileHeader.Name = zipEntryName

		// File info for the binary executable
		binaryWriter, binaryWriterErr := zipWriter.CreateHeader(fileHeader)
		if binaryWriterErr != nil {
			return binaryWriterErr
		}
		reader, readerErr := os.Open(fullPathSource)
		if readerErr != nil {
			return readerErr
		}
		written, copyErr := io.Copy(binaryWriter, reader)
		reader.Close()

		logger.WithFields(logrus.Fields{
			"WrittenBytes": written,
			"SourcePath":   fullPathSource,
			"ZipName":      zipEntryName,
		}).Debug("Archiving file")
		return copyErr
	}

	directoryWalker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		// Normalize the Name
		platformName := strings.TrimPrefix(strings.TrimPrefix(path, rootSource), string(os.PathSeparator))
		header.Name = linuxZipName(platformName)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
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
		_, err = io.Copy(writer, file)
		return err
	}

	fileInfo, err := os.Stat(fullPathSource)
	if nil != err {
		return err
	}
	switch mode := fileInfo.Mode(); {
	case mode.IsDir():
		err = filepath.Walk(fullPathSource, directoryWalker)
	case mode.IsRegular():
		err = appendFile(fileInfo)
	default:
		err = errors.New("Inavlid source type")
	}
	return err
}
