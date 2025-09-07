package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ChunkInfo struct {
	Index    int
	Start    int64
	End      int64
	Progress float64
	Status   string
}

type DownloadTask struct {
	ID             string
	URL            string
	OutputFile     string
	TotalSize      int64
	Downloaded     int64
	ChunkCount     int
	Chunks         []ChunkInfo
	Status         string
	Progress       float64
	Speed          float64
	StartTime      time.Time
	MD5Hash        string
	SHA256Hash     string
	progressBar    *widget.ProgressBar
	statusLabel    *widget.Label
	speedLabel     *widget.Label
	fileNameLabel  *widget.Label
	actionButton   *widget.Button
	container      *fyne.Container
	chunkBars      []*widget.ProgressBar
	chunkContainer *fyne.Container
	mu             sync.Mutex
	cancelFunc     func()
}

type Downloader struct {
	app           fyne.App
	window        fyne.Window
	tasks         map[string]*DownloadTask
	taskContainer *container.Scroll
	taskList      *fyne.Container
	urlEntry      *widget.Entry
	addButton     *widget.Button
	clearButton   *widget.Button
	statsLabel    *widget.Label
	outputFolder  string
	chunkCount    int
	mu            sync.Mutex
}

func NewDownloader() *Downloader {
	myApp := app.NewWithID("com.chunkeddownloader.app")
	myApp.Settings().SetTheme(&myTheme{})

	myWindow := myApp.NewWindow("Advanced Download Manager")
	myWindow.Resize(fyne.NewSize(1000, 700))

	// Get default download folder
	homeDir, _ := os.UserHomeDir()
	defaultOutput := filepath.Join(homeDir, "Downloads")

	d := &Downloader{
		app:          myApp,
		window:       myWindow,
		tasks:        make(map[string]*DownloadTask),
		outputFolder: defaultOutput,
		chunkCount:   10, // Default 10 chunks
	}

	// Load saved settings
	d.loadSettings()

	myWindow.SetContent(d.createUI())
	myWindow.CenterOnScreen()

	return d
}

func (d *Downloader) createUI() fyne.CanvasObject {
	// URL input section
	d.urlEntry = widget.NewEntry()
	d.urlEntry.SetPlaceHolder("Enter download URL (supports multiple formats)")

	d.addButton = widget.NewButtonWithIcon("Add Download", theme.DownloadIcon(), d.addDownload)
	d.addButton.Importance = widget.HighImportance

	// Icon-only control buttons
	d.clearButton = widget.NewButtonWithIcon("", theme.DeleteIcon(), d.clearCompleted)
	d.clearButton.Importance = widget.LowImportance

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), d.showSettings)
	settingsBtn.Importance = widget.LowImportance

	// Group all buttons together
	buttonGroup := container.NewHBox(
		d.addButton,
		d.clearButton,
		settingsBtn,
	)

	inputSection := container.NewBorder(
		nil, nil,
		widget.NewLabel("URL:"),
		buttonGroup,
		d.urlEntry,
	)

	// Stats section - single line with stats on left, settings on right
	d.statsLabel = widget.NewLabel("Active: 0 | Completed: 0 | Failed: 0")
	settingsLabel := widget.NewLabel(fmt.Sprintf("Output: %s | Chunks: %d",
		truncateString(d.outputFolder, 30), d.chunkCount))
	settingsLabel.TextStyle.Italic = true

	statsSection := container.NewBorder(
		nil, nil,
		d.statsLabel,
		settingsLabel,
		nil,
	)

	// Download tasks container
	d.taskList = container.NewVBox()
	d.taskContainer = container.NewScroll(d.taskList)
	d.taskContainer.SetMinSize(fyne.NewSize(950, 400))

	// Main layout with clean white design
	content := container.NewBorder(
		container.NewVBox(
			container.NewPadded(inputSection),
			statsSection,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewPadded(d.taskContainer),
	)

	return content
}

func (d *Downloader) addDownload() {
	urlStr := d.urlEntry.Text
	if urlStr == "" {
		dialog.ShowError(fmt.Errorf("Please enter a URL"), d.window)
		return
	}

	// Validate URL
	if _, err := url.Parse(urlStr); err != nil {
		dialog.ShowError(fmt.Errorf("Invalid URL: %v", err), d.window)
		return
	}

	// Create new download task
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	task := &DownloadTask{
		ID:         taskID,
		URL:        urlStr,
		Status:     "Preparing...",
		ChunkCount: d.chunkCount,
		StartTime:  time.Now(),
	}

	// Create UI for task
	task.createTaskUI(d)

	// Add to tasks
	d.mu.Lock()
	d.tasks[taskID] = task
	d.mu.Unlock()

	// Add to UI
	d.taskList.Add(task.container)

	// Clear entry
	d.urlEntry.SetText("")

	// Start download
	go d.startDownload(task)

	// Update stats
	d.updateStats()
}

func (task *DownloadTask) createTaskUI(d *Downloader) {
	// Modern card design with better layout
	cardBg := canvas.NewRectangle(color.NRGBA{R: 250, G: 250, B: 252, A: 255})
	cardBg.SetMinSize(fyne.NewSize(920, 80))
	cardBg.StrokeColor = color.NRGBA{R: 220, G: 220, B: 225, A: 255}
	cardBg.StrokeWidth = 1
	cardBg.CornerRadius = 6

	// File name and copy button
	task.fileNameLabel = widget.NewLabelWithStyle("Preparing download...",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Copy URL button
	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		// Copy URL to clipboard
		d.window.Clipboard().SetContent(task.URL)
	})
	copyBtn.Importance = widget.LowImportance
	copyBtn.Resize(fyne.NewSize(30, 30))

	// Main progress bar
	task.progressBar = widget.NewProgressBar()
	task.progressBar.SetValue(0)

	// Status display - prominent status indicator
	task.statusLabel = widget.NewLabelWithStyle("Preparing...",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	task.speedLabel = widget.NewLabel("-- MB/s")

	// Action buttons - moved to bottom left
	task.actionButton = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
		if task.Status == "Downloading" {
			task.Status = "Paused"
			task.actionButton.SetIcon(theme.MediaPlayIcon())
		} else if task.Status == "Paused" {
			task.Status = "Downloading"
			task.actionButton.SetIcon(theme.MediaPauseIcon())
		} else if task.Status == "Failed" {
			// Retry failed download
			task.Status = "Preparing..."
			task.actionButton.SetIcon(theme.MediaPauseIcon())
			go d.startDownload(task)
		}
	})
	task.actionButton.Importance = widget.LowImportance

	removeBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		// Remove task immediately
		d.removeTask(task)
	})
	removeBtn.Importance = widget.LowImportance

	// Layout: File name and copy button on top
	fileInfo := container.NewHBox(
		task.fileNameLabel,
		layout.NewSpacer(),
		copyBtn,
	)

	// Status, chunks, and speed on bottom right
	statusInfo := container.NewHBox(
		layout.NewSpacer(),
		task.statusLabel,
		widget.NewLabel("|"),
		task.speedLabel,
	)

	// Action buttons on bottom left
	actions := container.NewHBox(
		task.actionButton,
		removeBtn,
	)

	// Main content layout
	mainContent := container.NewBorder(
		fileInfo,
		container.NewHBox(actions, layout.NewSpacer(), statusInfo),
		nil,
		nil,
		task.progressBar,
	)

	// Add padding to the card
	paddedContent := container.NewPadded(mainContent)
	task.container = container.NewStack(cardBg, paddedContent)
	task.container = container.NewPadded(task.container)
}

func (d *Downloader) startDownload(task *DownloadTask) {
	// Get file info
	err := d.getFileInfo(task)
	if err != nil {
		task.Status = "Failed"
		fyne.Do(func() {
			task.updateStatusDisplay()
			task.actionButton.SetIcon(theme.ViewRefreshIcon())
		})
		return
	}

	// Update UI with file info
	fileName := filepath.Base(task.OutputFile)
	fyne.Do(func() {
		// Update file name directly using stored reference
		task.fileNameLabel.SetText(fileName)
		// Update status with chunk info
		task.updateStatusDisplay()
	})

	// Initialize chunks
	d.initializeChunks(task)

	// Start downloading
	task.Status = "Downloading"
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Limit concurrent chunks

	for i := range task.Chunks {
		wg.Add(1)
		go func(chunk *ChunkInfo) {
			defer wg.Done()
			semaphore <- struct{}{}
			d.downloadChunk(task, chunk)
			<-semaphore
		}(&task.Chunks[i])
		time.Sleep(200 * time.Millisecond)
	}

	// Monitor progress
	go d.monitorProgress(task)

	// Wait for completion
	wg.Wait()

	// Check results
	successCount := 0
	for _, chunk := range task.Chunks {
		if chunk.Status == "Completed" {
			successCount++
		}
	}

	if successCount < len(task.Chunks)/2 {
		// Fallback to single download
		d.downloadSingleFile(task)
		return
	}

	if successCount == len(task.Chunks) {
		// Merge chunks
		err = d.mergeChunks(task)
		if err != nil {
			task.Status = "Failed"
			fyne.Do(func() {
				task.updateStatusDisplay()
				task.actionButton.SetIcon(theme.ViewRefreshIcon())
			})
			return
		}

		// Calculate checksums
		d.calculateChecksums(task)

		task.Status = "Completed"
		fyne.Do(func() {
			task.progressBar.SetValue(1.0)
			task.updateStatusDisplay()
			task.actionButton.SetIcon(theme.FolderOpenIcon())
			// Create a new closure to capture the downloader reference
			localD := d
			task.actionButton.OnTapped = func() {
				// Open file location
				localD.openFileLocation(task.OutputFile)
			}
		})
	} else {
		task.Status = "Failed"
		fyne.Do(func() {
			task.updateStatusDisplay()
			task.actionButton.SetIcon(theme.ViewRefreshIcon())
		})
	}

	d.updateStats()
}

func (d *Downloader) getFileInfo(task *DownloadTask) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	req, err := http.NewRequest("HEAD", task.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Try GET with Range
		req, _ = http.NewRequest("GET", task.URL, nil)
		req.Header.Set("Range", "bytes=0-0")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		resp, err = client.Do(req)
		if err != nil {
			return err
		}
	}
	defer resp.Body.Close()

	// Get size
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		task.TotalSize, _ = strconv.ParseInt(cl, 10, 64)
	} else if cr := resp.Header.Get("Content-Range"); cr != "" {
		parts := strings.Split(cr, "/")
		if len(parts) == 2 {
			task.TotalSize, _ = strconv.ParseInt(parts[1], 10, 64)
		}
	}

	// Get filename
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if idx := strings.Index(cd, "filename="); idx != -1 {
			task.OutputFile = strings.Trim(cd[idx+9:], "\"")
		}
	} else {
		task.OutputFile = path.Base(task.URL)
		if task.OutputFile == "/" || task.OutputFile == "." {
			task.OutputFile = "download_" + task.ID
		}
	}

	// Set full path with output folder
	task.OutputFile = filepath.Join(d.outputFolder, task.OutputFile)

	// Ensure output folder exists
	if err := os.MkdirAll(d.outputFolder, 0755); err != nil {
		return fmt.Errorf("failed to create output folder: %v", err)
	}

	return nil
}

func (d *Downloader) initializeChunks(task *DownloadTask) {
	chunkSize := task.TotalSize / int64(task.ChunkCount)
	task.Chunks = make([]ChunkInfo, task.ChunkCount)

	for i := 0; i < task.ChunkCount; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == task.ChunkCount-1 {
			end = task.TotalSize - 1
		}

		task.Chunks[i] = ChunkInfo{
			Index:  i,
			Start:  start,
			End:    end,
			Status: "Pending",
		}
	}
}

func (d *Downloader) downloadChunk(task *DownloadTask, chunk *ChunkInfo) {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", task.URL, nil)
	if err != nil {
		chunk.Status = "Failed"
		return
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", chunk.Start, chunk.End))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		chunk.Status = "Failed"
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		chunk.Status = "Failed"
		return
	}

	tempFile := fmt.Sprintf("%s.part%d", task.OutputFile, chunk.Index)
	file, err := os.Create(tempFile)
	if err != nil {
		chunk.Status = "Failed"
		return
	}
	defer file.Close()

	buffer := make([]byte, 32*1024)
	totalBytes := chunk.End - chunk.Start + 1
	downloaded := int64(0)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			file.Write(buffer[:n])
			downloaded += int64(n)
			chunk.Progress = float64(downloaded) / float64(totalBytes)

			task.mu.Lock()
			task.Downloaded += int64(n)
			task.mu.Unlock()

			// Update progress in UI thread
			fyne.Do(func() {
				task.updateStatusDisplay()
			})
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			chunk.Status = "Failed"
			return
		}

		if task.Status == "Cancelled" {
			return
		}
	}

	chunk.Status = "Completed"
	chunk.Progress = 1.0

	// Update status when chunk completes
	fyne.Do(func() {
		task.updateStatusDisplay()
	})
}

func (d *Downloader) downloadSingleFile(task *DownloadTask) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", task.URL, nil)
	if err != nil {
		task.Status = "Failed"
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		task.Status = "Failed"
		return
	}
	defer resp.Body.Close()

	file, err := os.Create(task.OutputFile)
	if err != nil {
		task.Status = "Failed"
		return
	}
	defer file.Close()

	buffer := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			file.Write(buffer[:n])
			task.Downloaded += int64(n)
			task.Progress = float64(task.Downloaded) / float64(task.TotalSize)
			fyne.Do(func() {
				task.progressBar.SetValue(task.Progress)
			})
		}

		if err == io.EOF {
			break
		}
	}

	task.Status = "Completed"
}

func (d *Downloader) mergeChunks(task *DownloadTask) error {
	outputFile, err := os.Create(task.OutputFile)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	for i := 0; i < task.ChunkCount; i++ {
		tempFile := fmt.Sprintf("%s.part%d", task.OutputFile, i)
		input, err := os.Open(tempFile)
		if err != nil {
			continue
		}

		io.Copy(outputFile, input)
		input.Close()
		os.Remove(tempFile)
	}

	return nil
}

func (task *DownloadTask) updateStatusDisplay() {
	// Update status display based on current status with chunk info
	statusText := ""
	switch task.Status {
	case "Downloading":
		statusText = "Downloading..."
	case "Paused":
		statusText = "Paused"
	case "Completed":
		statusText = "Completed"
	case "Failed":
		statusText = "Failed"
	case "Preparing...":
		statusText = "Preparing..."
	case "Cancelled":
		statusText = "Cancelled"
	default:
		statusText = task.Status
	}

	// Add chunk count if available
	if task.ChunkCount > 0 {
		statusText = fmt.Sprintf("%s | %d chunks", statusText, task.ChunkCount)
	}

	task.statusLabel.SetText(statusText)
}

func (d *Downloader) calculateChecksums(task *DownloadTask) {
	file, err := os.Open(task.OutputFile)
	if err != nil {
		return
	}
	defer file.Close()

	md5Hash := md5.New()
	sha256Hash := sha256.New()

	if _, err := io.Copy(io.MultiWriter(md5Hash, sha256Hash), file); err == nil {
		task.MD5Hash = hex.EncodeToString(md5Hash.Sum(nil))
		task.SHA256Hash = hex.EncodeToString(sha256Hash.Sum(nil))
	}
}

func (d *Downloader) monitorProgress(task *DownloadTask) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastDownloaded := int64(0)

	for range ticker.C {
		if task.Status != "Downloading" {
			break
		}

		task.mu.Lock()
		currentDownloaded := task.Downloaded
		task.mu.Unlock()

		// Calculate speed
		speed := float64(currentDownloaded-lastDownloaded) * 2 / (1024 * 1024) // MB/s
		lastDownloaded = currentDownloaded

		// Update progress
		if task.TotalSize > 0 {
			progress := float64(currentDownloaded) / float64(task.TotalSize)
			fyne.Do(func() {
				task.progressBar.SetValue(progress)
				// Show file size and speed
				fileSizeMB := float64(task.TotalSize) / (1024 * 1024)
				task.speedLabel.SetText(fmt.Sprintf("%.1f MB | %.2f MB/s", fileSizeMB, speed))

				// Update status display
				task.updateStatusDisplay()
			})
		}
	}
}

func (d *Downloader) removeTask(task *DownloadTask) {
	// Cancel if still downloading
	if task.Status == "Downloading" || task.Status == "Preparing..." {
		task.Status = "Cancelled"
		if task.cancelFunc != nil {
			task.cancelFunc()
		}
	}

	// Remove from tasks map
	d.mu.Lock()
	delete(d.tasks, task.ID)
	d.mu.Unlock()

	// Remove from UI
	fyne.Do(func() {
		d.taskList.Remove(task.container)
		d.taskList.Refresh()
		d.taskContainer.Refresh()
	})

	d.updateStats()
}

func (d *Downloader) clearCompleted() {
	d.mu.Lock()
	toRemove := []*DownloadTask{}
	for _, task := range d.tasks {
		if task.Status == "Completed" || task.Status == "Cancelled" || task.Status == "Failed" {
			toRemove = append(toRemove, task)
		}
	}
	d.mu.Unlock()

	// Remove each task
	for _, task := range toRemove {
		d.removeTask(task)
	}
}

func (d *Downloader) updateStats() {
	active := 0
	completed := 0
	failed := 0

	d.mu.Lock()
	for _, task := range d.tasks {
		switch task.Status {
		case "Downloading", "Preparing...":
			active++
		case "Completed":
			completed++
		case "Failed":
			failed++
		}
	}
	d.mu.Unlock()

	fyne.Do(func() {
		d.statsLabel.SetText(fmt.Sprintf("Active: %d | Completed: %d | Failed: %d",
			active, completed, failed))
	})
}

func (d *Downloader) showSettings() {
	// Create settings dialog
	outputEntry := widget.NewEntry()
	outputEntry.SetText(d.outputFolder)
	outputEntry.Disable()

	browseBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				outputEntry.SetText(uri.Path())
			}
		}, d.window)
	})

	outputRow := container.NewBorder(nil, nil, nil, browseBtn, outputEntry)

	// Chunk count slider
	chunkSlider := widget.NewSlider(1, 50)
	chunkSlider.Value = float64(d.chunkCount)
	chunkSlider.Step = 1

	chunkLabel := widget.NewLabel(fmt.Sprintf("Number of chunks: %d", d.chunkCount))

	chunkSlider.OnChanged = func(value float64) {
		chunkLabel.SetText(fmt.Sprintf("Number of chunks: %d", int(value)))
	}

	// Create form
	content := container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewVBox(
			widget.NewLabel("Download Folder:"),
			outputRow,
		),
		widget.NewSeparator(),
		container.NewVBox(
			chunkLabel,
			chunkSlider,
		),
		widget.NewSeparator(),
		widget.NewLabel("Note: Changes apply to new downloads only"),
	)

	// Create custom dialog
	settingsDialog := dialog.NewCustomConfirm("Settings", "Save", "Cancel", content, func(save bool) {
		if save {
			d.outputFolder = outputEntry.Text
			d.chunkCount = int(chunkSlider.Value)
			d.saveSettings()
		}
	}, d.window)

	settingsDialog.Resize(fyne.NewSize(500, 300))
	settingsDialog.Show()
}

func (d *Downloader) openFileLocation(filePath string) {
	// Get the directory containing the file
	dir := filepath.Dir(filePath)

	// Platform specific file manager opening
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "linux":
		cmd = exec.Command("xdg-open", dir)
	default:
		dialog.ShowInformation("File Location",
			fmt.Sprintf("File saved to:\n%s", filePath), d.window)
		return
	}

	err := cmd.Start()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to open folder: %v", err), d.window)
	}
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Custom theme for modern look
type myTheme struct{}

func (m *myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // Pure white background
	case theme.ColorNameForeground:
		return color.NRGBA{R: 33, G: 37, B: 41, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0, G: 122, B: 255, A: 255} // Blue accent
	case theme.ColorNameButton:
		return color.NRGBA{R: 240, G: 240, B: 240, A: 255} // Light gray buttons
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 250, G: 250, B: 250, A: 255} // Light input background
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 150, G: 150, B: 150, A: 255}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (m *myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m *myTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNameScrollBar:
		return 16
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func (d *Downloader) saveSettings() {
	prefs := d.app.Preferences()
	prefs.SetString("outputFolder", d.outputFolder)
	prefs.SetInt("chunkCount", d.chunkCount)
}

func (d *Downloader) loadSettings() {
	prefs := d.app.Preferences()

	if folder := prefs.String("outputFolder"); folder != "" {
		d.outputFolder = folder
	}

	if chunks := prefs.Int("chunkCount"); chunks > 0 {
		d.chunkCount = chunks
	}
}

func main() {
	downloader := NewDownloader()
	downloader.window.ShowAndRun()
}
