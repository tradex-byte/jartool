package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type ConsoleLogger struct{}

type DownloadProgress struct {
	TotalBytes   int64
	WrittenBytes int64
	Logger       *ConsoleLogger
}

func main() {
	decompileInput := flag.String("d", "", "")
	compileInput := flag.String("c", "", "")
	outputLocation := flag.String("o", "", "")
	replaceDirectory := flag.String("r", "", "")
	searchString := flag.String("s", "", "")
	newString := flag.String("n", "", "")
	helpFlag := flag.Bool("h", false, "")
	flag.Parse()

	logger := &ConsoleLogger{}

	if *helpFlag {
		showHelpScreen()
		return
	}

	logger.writeStatus("Starting JAR Tool...")

	javaHome := locateJavaInstallation(logger)
	if javaHome == "" {
		logger.writeInfo("Java not found locally, initiating download...")
		javaHome = downloadJavaRuntime(logger)
		if javaHome == "" {
			logger.writeError("Failed to obtain Java runtime")
			os.Exit(1)
		}
	}

	jarCommandPath := javaHome + "/bin/jar"
	if runtime.GOOS == "windows" {
		jarCommandPath += ".exe"
	}

	if *decompileInput != "" && *outputLocation != "" {
		executeDecompile(*decompileInput, *outputLocation, jarCommandPath, logger)
	} else if *replaceDirectory != "" && *searchString != "" && *newString != "" {
		executeTextReplacement(*replaceDirectory, *searchString, *newString, logger)
	} else if *compileInput != "" && *outputLocation != "" {
		executeCompilation(*compileInput, *outputLocation, jarCommandPath, logger)
	}
}

func showHelpScreen() {
	fmt.Println("JAR Tool - Decompile & Compile JAR Files")
	fmt.Println("=========================================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  Decompile:      jartool -d input.jar -o output_dir")
	fmt.Println("  Replace:        jartool -r output_dir -s http://old.com -n http://new.com")
	fmt.Println("  Compile:        jartool -c input_dir -o output.jar")
	fmt.Println("  Help:           jartool -h")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  jartool -d app.jar -o decompiled")
	fmt.Println("  jartool -r decompiled -s http://127.0.0.1 -n https://example.com")
	fmt.Println("  jartool -c decompiled -o app_modified.jar")
}

func (l *ConsoleLogger) writeInfo(message string, arguments ...interface{}) {
	fmt.Printf("[INFO] "+message+"\n", arguments...)
}

func (l *ConsoleLogger) writeWarning(message string, arguments ...interface{}) {
	fmt.Printf("[WARN] "+message+"\n", arguments...)
}

func (l *ConsoleLogger) writeError(message string, arguments ...interface{}) {
	fmt.Printf("[ERROR] "+message+"\n", arguments...)
}

func (l *ConsoleLogger) writeStatus(message string, arguments ...interface{}) {
	fmt.Printf("[STATUS] "+message+"\n", arguments...)
}

func (l *ConsoleLogger) writeSuccess(message string, arguments ...interface{}) {
	fmt.Printf("[SUCCESS] "+message+"\n", arguments...)
}

func (l *ConsoleLogger) writeProgress(message string, arguments ...interface{}) {
	fmt.Printf("[PROGRESS] "+message+"\n", arguments...)
}

func locateJavaInstallation(logger *ConsoleLogger) string {
	executableDirectory, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	javaDirectory := executableDirectory + "/java"

	javaDirectories, _ := filepath.Glob(javaDirectory + "/jdk*")
	for _, javaPath := range javaDirectories {
		jarExecutable := javaPath + "/bin/jar"
		if runtime.GOOS == "windows" {
			jarExecutable += ".exe"
		}
		if _, errorStat := os.Stat(jarExecutable); errorStat == nil {
			logger.writeSuccess("Java located at: %s", javaPath)
			return javaPath
		}
	}

	logger.writeInfo("No Java installation found in local directory")
	return ""
}

func downloadJavaRuntime(logger *ConsoleLogger) string {
	executableDirectory, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	javaDirectory := executableDirectory + "/java"

	logger.writeInfo("Creating Java directory: %s", javaDirectory)
	os.MkdirAll(javaDirectory, 0)

	downloadURL := ""
	archiveFileName := ""
	switch runtime.GOOS {
	case "windows":
		downloadURL = "https://github.com/adoptium/temurin8-binaries/releases/download/jdk8u502-b07/OpenJDK8U-jdk_x64_windows_hotspot_8u502b07.zip"
		archiveFileName = javaDirectory + "/jdk.zip"
		logger.writeInfo("Detected Windows operating system")
	case "linux":
		downloadURL = "https://github.com/adoptium/temurin8-binaries/releases/download/jdk8u502-b07/OpenJDK8U-jdk_x64_linux_hotspot_8u502b07.tar.gz"
		archiveFileName = javaDirectory + "/jdk.tar.gz"
		logger.writeInfo("Detected Linux operating system")
	case "darwin":
		downloadURL = "https://github.com/adoptium/temurin8-binaries/releases/download/jdk8u502-b07/OpenJDK8U-jdk_x64_mac_hotspot_8u502b07.tar.gz"
		archiveFileName = javaDirectory + "/jdk.tar.gz"
		logger.writeInfo("Detected macOS operating system")
	default:
		logger.writeError("Unsupported operating system: %s", runtime.GOOS)
		return ""
	}

	logger.writeStatus("Downloading Java runtime...")

	httpResponse, errorGet := http.Get(downloadURL)
	if errorGet != nil {
		logger.writeError("Download failed: %v", errorGet)
		return ""
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != 200 {
		logger.writeError("HTTP error: %s", httpResponse.Status)
		return ""
	}

	fileSize := httpResponse.ContentLength

	outputFile, errorCreate := os.Create(archiveFileName)
	if errorCreate != nil {
		logger.writeError("Failed to create file: %v", errorCreate)
		return ""
	}
	defer outputFile.Close()

	progressTracker := &DownloadProgress{
		TotalBytes:   fileSize,
		WrittenBytes: 0,
		Logger:       logger,
	}
	io.Copy(outputFile, io.TeeReader(httpResponse.Body, progressTracker))
	outputFile.Close()

	logger.writeStatus("Extracting Java runtime...")
	if strings.HasSuffix(archiveFileName, ".zip") {
		extractZipArchive(archiveFileName, javaDirectory, logger)
	} else {
		extractCommand := exec.Command("tar", "-xzf", archiveFileName, "-C", javaDirectory)
		extractCommand.Run()
	}

	os.Remove(archiveFileName)

	javaDirectories, _ := filepath.Glob(javaDirectory + "/jdk*")
	if len(javaDirectories) > 0 {
		logger.writeSuccess("Java installed successfully at: %s", javaDirectories[0])
		return javaDirectories[0]
	}

	logger.writeError("Installation failed - no JDK directory found")
	return ""
}

func (p *DownloadProgress) Write(data []byte) (int, error) {
	bytesWritten := len(data)
	p.WrittenBytes += int64(bytesWritten)
	percentComplete := float64(p.WrittenBytes) / float64(p.TotalBytes) * 100
	if int(percentComplete)%10 == 0 || p.WrittenBytes == p.TotalBytes {
		p.Logger.writeProgress("Downloading: %.1f%% (%d/%d MB)",
			percentComplete,
			p.WrittenBytes/1024/1024,
			p.TotalBytes/1024/1024)
	}
	return bytesWritten, nil
}

func extractZipArchive(archivePath, destinationPath string, logger *ConsoleLogger) {
	zipReader, errorOpen := zip.OpenReader(archivePath)
	if errorOpen != nil {
		logger.writeError("Failed to open ZIP: %v", errorOpen)
		return
	}
	defer zipReader.Close()

	totalFiles := len(zipReader.File)

	for fileIndex, zipFile := range zipReader.File {
		extractedPath := destinationPath + "/" + zipFile.Name
		if zipFile.FileInfo().IsDir() {
			os.MkdirAll(extractedPath, 0)
			continue
		}
		os.MkdirAll(filepath.Dir(extractedPath), 0)
		sourceFile, errorOpen := zipFile.Open()
		if errorOpen != nil {
			logger.writeWarning("Failed to open file in ZIP: %s", zipFile.Name)
			continue
		}
		destinationFile, errorCreate := os.Create(extractedPath)
		if errorCreate != nil {
			logger.writeWarning("Failed to create file: %s", extractedPath)
			sourceFile.Close()
			continue
		}
		io.Copy(destinationFile, sourceFile)
		sourceFile.Close()
		destinationFile.Close()

		if fileIndex%100 == 0 {
			logger.writeProgress("Extracted %d/%d files", fileIndex, totalFiles)
		}
	}
}

func executeDecompile(jarFilePath, outputDirectory, jarCommand string, logger *ConsoleLogger) {
	logger.writeStatus("Starting decompilation process...")
	logger.writeInfo("Source JAR: %s", jarFilePath)
	logger.writeInfo("Output directory: %s", outputDirectory)

	if _, errorStat := os.Stat(jarFilePath); errorStat != nil {
		logger.writeError("JAR file not found: %s", jarFilePath)
		return
	}

	logger.writeInfo("Creating output directory")
	os.MkdirAll(outputDirectory, 0)

	absoluteJarPath, _ := filepath.Abs(jarFilePath)
	absoluteOutputPath, _ := filepath.Abs(outputDirectory)

	currentDirectory, _ := os.Getwd()
	os.Chdir(absoluteOutputPath)
	defer os.Chdir(currentDirectory)

	logger.writeStatus("Executing jar extraction...")

	extractCommand := exec.Command(jarCommand, "xf", absoluteJarPath)
	commandOutput, errorRun := extractCommand.CombinedOutput()
	if errorRun != nil {
		logger.writeError("Decompilation failed: %v", errorRun)
		if len(commandOutput) > 0 {
			logger.writeError("Command output: %s", string(commandOutput))
		}
		return
	}

	logger.writeSuccess("Decompilation completed successfully")
	logger.writeInfo("Files extracted to: %s", absoluteOutputPath)

	extractedFileCount := 0
	filepath.Walk(absoluteOutputPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			extractedFileCount++
		}
		return nil
	})
	logger.writeInfo("Extracted %d files", extractedFileCount)
}

func executeTextReplacement(directoryPath, searchText, replaceText string, logger *ConsoleLogger) {
	logger.writeStatus("Starting text replacement operation...")
	logger.writeInfo("Target directory: %s", directoryPath)
	logger.writeInfo("Search pattern: %s", searchText)
	logger.writeInfo("Replace with: %s", replaceText)

	if _, errorStat := os.Stat(directoryPath); errorStat != nil {
		logger.writeError("Directory not found: %s", directoryPath)
		return
	}

	totalFilesScanned := 0
	filesModified := 0

	filepath.Walk(directoryPath, func(filePath string, fileInfo os.FileInfo, walkError error) error {
		if walkError != nil {
			logger.writeWarning("Error accessing %s: %v", filePath, walkError)
			return nil
		}
		if fileInfo.IsDir() {
			return nil
		}

		totalFilesScanned++

		fileContent, errorRead := os.Open(filePath)
		if errorRead != nil {
			logger.writeWarning("Failed to read %s: %v", filepath.Base(filePath), errorRead)
			return nil
		}
		defer fileContent.Close()

		contentBytes := make([]byte, 0)
		buffer := make([]byte, 1024)
		for {
			n, err := fileContent.Read(buffer)
			if n > 0 {
				contentBytes = append(contentBytes, buffer[:n]...)
			}
			if err != nil {
				break
			}
		}

		contentString := string(contentBytes)
		if strings.Contains(contentString, searchText) {
			updatedContent := strings.Replace(contentString, searchText, replaceText, -1)
			fileContent.Close()
			os.Remove(filePath)
			file, _ := os.Create(filePath)
			file.Write([]byte(updatedContent))
			file.Close()
			filesModified++
			logger.writeInfo("Modified: %s", filepath.Base(filePath))
		}
		return nil
	})

	logger.writeSuccess("Text replacement completed")
	logger.writeInfo("Scanned %d files", totalFilesScanned)
	logger.writeInfo("Modified %d files", filesModified)
}

func executeCompilation(sourceDirectory, outputJar, jarCommand string, logger *ConsoleLogger) {
	logger.writeStatus("Starting compilation process...")
	logger.writeInfo("Source directory: %s", sourceDirectory)
	logger.writeInfo("Output JAR: %s", outputJar)

	if _, errorStat := os.Stat(sourceDirectory); errorStat != nil {
		logger.writeError("Directory not found: %s", sourceDirectory)
		return
	}

	if _, errorStat := os.Stat(outputJar); errorStat == nil {
		logger.writeInfo("Removing existing JAR: %s", outputJar)
		os.Remove(outputJar)
	}

	absoluteSourcePath, _ := filepath.Abs(sourceDirectory)
	absoluteOutputPath, _ := filepath.Abs(outputJar)

	manifestFilePath := absoluteSourcePath + "/META-INF/MANIFEST.MF"
	var compileCommand *exec.Cmd

	if _, errorStat := os.Stat(manifestFilePath); errorStat == nil {
		logger.writeInfo("Using existing MANIFEST.MF")
		compileCommand = exec.Command(jarCommand, "cfm", absoluteOutputPath, manifestFilePath, "-C", absoluteSourcePath, ".")
	} else {
		logger.writeInfo("Creating default MANIFEST.MF")
		temporaryManifest := absoluteSourcePath + "/MANIFEST.MF"
		file, _ := os.Create(temporaryManifest)
		file.Write([]byte("Manifest-Version: 1.0\n"))
		file.Close()
		defer os.Remove(temporaryManifest)
		compileCommand = exec.Command(jarCommand, "cfm", absoluteOutputPath, temporaryManifest, "-C", absoluteSourcePath, ".")
	}

	logger.writeStatus("Executing jar creation...")

	commandOutput, errorRun := compileCommand.CombinedOutput()
	if errorRun != nil {
		logger.writeError("Compilation failed: %v", errorRun)
		if len(commandOutput) > 0 {
			logger.writeError("Command output: %s", string(commandOutput))
		}
		return
	}

	if fileInfo, errorStat := os.Stat(absoluteOutputPath); errorStat == nil {
		sizeKilobytes := float64(fileInfo.Size()) / 1024
		logger.writeSuccess("Compilation completed successfully")
		logger.writeInfo("JAR size: %.2f KB", sizeKilobytes)
		logger.writeInfo("Created: %s", absoluteOutputPath)
	} else {
		logger.writeError("JAR file was not created")
	}
}
