package main

//go:generate go-winres make

import (
	"embed"
	"fmt"
	"github.com/TheTitanrain/w32"
	"github.com/klauspost/compress/s2"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"path"
	"time"
	"unsafe"
	"vpvc-executable-wrapper/thirdparty/unzip"
)

var (
	//go:embed resources/app-archive.zip.s2
	res embed.FS
)

func DecodeS2Stream(src io.Reader, dst io.Writer) error {
	dec := s2.NewReader(src)
	_, err := io.Copy(dst, dec)
	return err
}

func main() {
	// Create invisible window so Windows makes
	// the application launched by this wrapper
	// start in foreground
	invisibleWrapperWindowHandle := w32.CreateWindowEx(
		w32.WS_EX_OVERLAPPEDWINDOW,
		windows.StringToUTF16Ptr(""),
		nil,
		w32.WS_OVERLAPPEDWINDOW,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		nil)

	userHomeDirectoryPath, userHomeDirectoryExtractionError := os.UserHomeDir()

	if userHomeDirectoryExtractionError != nil {
		w32.DestroyWindow(invisibleWrapperWindowHandle)

		fmt.Println(userHomeDirectoryExtractionError)
		return
	}

	// Always use the same directory path instead of
	// new temporary directories so Windows remembers
	// the application and does not show the
	// firewall prompt every time
	wrapperTempFilesDirectoryPath := path.Join(userHomeDirectoryPath, ".vpvc")

	appExtractionDirectoryPath := path.Join(wrapperTempFilesDirectoryPath, "app-extracted")

	// Make sure the app is not running already by checking if the
	// extracted app directory exists and deleting the main
	// executable fails (because it's open)
	appExtractionDirectoryContentsBeforeExtraction, appExtractionDirectoryContentListingBeforeExtractionError := os.ReadDir(appExtractionDirectoryPath)
	if appExtractionDirectoryContentListingBeforeExtractionError == nil {
		appExtractionDirectoryContentsFirstEntryBeforeExtraction := appExtractionDirectoryContentsBeforeExtraction[0]

		if appExtractionDirectoryContentsFirstEntryBeforeExtraction.IsDir() {
			appExecutablePath := path.Join(appExtractionDirectoryPath, appExtractionDirectoryContentsFirstEntryBeforeExtraction.Name(), "VPVC.exe")

			appExecutableDeletionError := os.Remove(appExecutablePath)

			if appExecutableDeletionError != nil {
				w32.DestroyWindow(invisibleWrapperWindowHandle)

				fmt.Println(appExecutableDeletionError)
				return
			}
		}
	}

	_ = os.RemoveAll(wrapperTempFilesDirectoryPath)
	_ = os.Mkdir(wrapperTempFilesDirectoryPath, os.ModePerm)

	zipArchiveTempFilePath := path.Join(wrapperTempFilesDirectoryPath, "app-archive")

	inputArchiveFile, inputArchiveOpeningError := res.Open("resources/app-archive.zip.s2")
	zipArchiveTempFile, zipArchiveTempFileCreationError := os.Create(zipArchiveTempFilePath)

	if inputArchiveOpeningError != nil || zipArchiveTempFileCreationError != nil {
		_ = inputArchiveFile.Close()
		w32.DestroyWindow(invisibleWrapperWindowHandle)

		fmt.Println(inputArchiveOpeningError)
		return
	}

	decodingError := DecodeS2Stream(inputArchiveFile, zipArchiveTempFile)

	_ = inputArchiveFile.Close()

	_ = zipArchiveTempFile.Close()

	if decodingError != nil {
		w32.DestroyWindow(invisibleWrapperWindowHandle)

		fmt.Println(decodingError)
		return
	}

	appExtractionDirectoryCreationError := os.Mkdir(
		appExtractionDirectoryPath,
		os.ModePerm)

	if appExtractionDirectoryCreationError != nil {
		w32.DestroyWindow(invisibleWrapperWindowHandle)

		fmt.Println(appExtractionDirectoryCreationError)
		return
	}

	uz := unzip.New()

	_, extractionError := uz.Extract(zipArchiveTempFile.Name(), appExtractionDirectoryPath)
	if extractionError != nil {
		w32.DestroyWindow(invisibleWrapperWindowHandle)

		fmt.Println(extractionError)
		return
	}

	os.Remove(zipArchiveTempFile.Name())

	appExtractionDirectoryContents, appExtractionDirectoryContentListingError := os.ReadDir(appExtractionDirectoryPath)

	if appExtractionDirectoryContentListingError != nil {
		_ = os.RemoveAll(wrapperTempFilesDirectoryPath)
		w32.DestroyWindow(invisibleWrapperWindowHandle)

		fmt.Println(appExtractionDirectoryContentListingError)
		return
	}

	appExtractionDirectoryContentsFirstEntry := appExtractionDirectoryContents[0]

	if !appExtractionDirectoryContentsFirstEntry.IsDir() {
		_ = os.RemoveAll(wrapperTempFilesDirectoryPath)
		w32.DestroyWindow(invisibleWrapperWindowHandle)
		return
	}

	appExecutablePath := path.Join(appExtractionDirectoryPath, appExtractionDirectoryContentsFirstEntry.Name(), "VPVC.exe")

	var appProcessInformation w32.PROCESS_INFORMATION
	appStartupInformation := &w32.STARTUPINFOW{}
	appStartingError := w32.CreateProcessW(
		appExecutablePath,
		"",
		nil,
		nil,
		0,
		0,
		unsafe.Pointer(nil),
		"",
		appStartupInformation,
		&appProcessInformation,
	)

	if appStartingError != nil {
		_ = os.RemoveAll(wrapperTempFilesDirectoryPath)
		w32.DestroyWindow(invisibleWrapperWindowHandle)

		fmt.Println(appStartingError)
		return
	}

	for true {
		hasAppProcessTerminated, appWaitingError := w32.WaitForSingleObject(appProcessInformation.Process, 0)

		if hasAppProcessTerminated {
			break
		}

		if appWaitingError != nil {
			_ = os.RemoveAll(wrapperTempFilesDirectoryPath)
			w32.DestroyWindow(invisibleWrapperWindowHandle)

			fmt.Println(appWaitingError)
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	_ = os.RemoveAll(wrapperTempFilesDirectoryPath)

	w32.DestroyWindow(invisibleWrapperWindowHandle)
}
