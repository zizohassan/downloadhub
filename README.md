# Download Manager

<img width="1002" height="732" alt="Screenshot 2025-09-08 at 12 04 31 AM" src="https://github.com/user-attachments/assets/3f7015b4-6601-45f8-971e-c2fa9e115b82" />

<img width="1002" height="731" alt="Screenshot 2025-09-08 at 12 04 51 AM" src="https://github.com/user-attachments/assets/60acab37-73af-42f4-911e-cbc1eb865eb4" />

<img width="502" height="399" alt="Screenshot 2025-09-08 at 12 05 06 AM" src="https://github.com/user-attachments/assets/e1fa5c1a-0e99-40a7-b66d-88b477ca6429" />


A modern, multi-threaded download manager built with Go and Fyne GUI framework. Features chunked downloads, progress tracking, and a clean, intuitive interface.



![Download Manager](https://img.shields.io/badge/Platform-macOS%20ARM64-blue)
![Go Version](https://img.shields.io/badge/Go-1.21+-green)
![License](https://img.shields.io/badge/License-MIT-yellow)

## ✨ Features

### 🚀 **Multi-threaded Downloads**
- **Chunked Downloads**: Split large files into multiple chunks for faster downloads
- **Configurable Chunks**: Choose from 1-50 chunks (default: 10)
- **Concurrent Processing**: Download multiple chunks simultaneously
- **Automatic Fallback**: Falls back to single-threaded download if chunked fails

### 📊 **Progress Tracking**
- **Real-time Progress**: Visual progress bars for each download
- **Speed Monitoring**: Live download speed display
- **File Information**: Shows file size, chunk count, and download status
- **Status Indicators**: Clear status display (Downloading, Completed, Failed, etc.)

### 🎨 **Modern Interface**
- **Clean Design**: Minimalist, professional UI
- **Custom Icon**: Beautiful download cloud icon
- **Responsive Layout**: Adapts to different window sizes
- **Dark/Light Theme**: Automatic theme adaptation

### ⚙️ **Settings & Configuration**
- **Output Folder**: Choose download destination
- **Chunk Count**: Configure number of download chunks
- **Persistent Settings**: Settings saved between sessions
- **Cross-platform**: Works on macOS, Windows, and Linux

### 🔧 **Advanced Features**
- **URL Validation**: Automatic URL format checking
- **File Integrity**: MD5 and SHA256 checksums
- **Resume Support**: Pause and resume downloads
- **Error Handling**: Robust error recovery and retry
- **Copy URL**: Easy URL copying to clipboard

## 📦 Installation

### macOS (Apple Silicon)
1. Download `Download-Manager-macOS-ARM64-with-custom-icon.zip`
2. Extract the ZIP file
3. Drag `Download Manager.app` to your Applications folder
4. Launch from Applications or double-click the app

### From Source
```bash
# Clone the repository
git clone <repository-url>
cd download-manager

# Install dependencies
go mod tidy

# Run the application
go run main.go

# Build for your platform
go build -o download-manager main.go
```

## 🚀 Usage

### Adding Downloads
1. **Enter URL**: Paste the download URL in the input field
2. **Click Add**: Press "Add Download" or hit Enter
3. **Monitor Progress**: Watch the progress bar and status updates
4. **Manage Downloads**: Use pause/resume, retry, or remove buttons

### Settings Configuration
1. **Open Settings**: Click the gear icon (⚙️)
2. **Choose Folder**: Select your preferred download directory
3. **Set Chunks**: Adjust the number of download chunks (1-50)
4. **Save**: Click "Save" to apply changes

### Managing Downloads
- **Pause/Resume**: Click the pause/play button on active downloads
- **Retry Failed**: Click the refresh button on failed downloads
- **Remove**: Click the trash icon to remove downloads
- **Clear Completed**: Use "Clear Completed" to remove finished downloads
- **Open File**: Click the folder icon to open the download location

## 🎯 Key Features Explained

### Chunked Downloads
The download manager splits large files into multiple chunks and downloads them simultaneously. This approach:
- **Increases Speed**: Parallel downloads are often faster than sequential
- **Improves Reliability**: If one chunk fails, others can continue
- **Better Resource Usage**: More efficient use of available bandwidth

### Progress Tracking
Each download shows:
- **Main Progress Bar**: Overall download progress (0-100%)
- **File Information**: Filename, size, and chunk count
- **Status Display**: Current state (Downloading, Completed, Failed, etc.)
- **Speed Information**: Real-time download speed and file size

### Settings Management
- **Output Folder**: Choose where files are saved
- **Chunk Count**: Configure parallel download threads
- **Persistent Storage**: Settings are saved automatically
- **Cross-Session**: Settings persist between app restarts

## 🛠️ Technical Details

### Built With
- **Go 1.21+**: Core programming language
- **Fyne v2**: Cross-platform GUI framework
- **HTTP/HTTPS**: Network protocols
- **Concurrent Processing**: Goroutines for parallel downloads

### Architecture
- **Multi-threaded**: Uses Go goroutines for concurrent operations
- **Event-driven**: Fyne framework handles UI events
- **Thread-safe**: Proper synchronization for UI updates
- **Modular Design**: Clean separation of concerns

### File Structure
```
download-manager/
├── main.go                    # Main application code
├── go.mod                     # Go module dependencies
├── go.sum                     # Dependency checksums
├── README.md                  # This file
├── download-cloud-svgrepo-com.svg  # App icon source
└── Download Manager.app/      # macOS application bundle
```

## 🔧 Configuration

### Default Settings
- **Chunk Count**: 10 chunks
- **Output Folder**: ~/Downloads
- **Concurrent Chunks**: 3 (limited for stability)
- **Timeout**: 30 seconds for requests

### Customization
You can modify these settings through the Settings dialog:
- **Chunks**: 1-50 (higher for faster connections)
- **Output**: Any accessible folder
- **Theme**: Follows system theme

## 🐛 Troubleshooting

### Common Issues

**App won't start on macOS**
- Right-click the app and select "Open"
- Go to System Preferences > Security & Privacy
- Allow the app to run

**Downloads are slow**
- Increase chunk count in Settings
- Check your internet connection
- Some servers may limit concurrent connections

**Downloads fail**
- Check the URL is valid and accessible
- Try reducing chunk count
- Check available disk space

**Settings not saving**
- Ensure the app has write permissions
- Check available disk space
- Restart the application

### Performance Tips
- **Optimal Chunks**: Start with 10-20 chunks
- **Fast Connections**: Use more chunks (20-50)
- **Slow Connections**: Use fewer chunks (5-10)
- **Large Files**: More chunks generally help
- **Small Files**: Fewer chunks may be better

## 📋 Requirements

### System Requirements
- **macOS**: 10.15+ (Apple Silicon recommended)
- **RAM**: 512MB minimum
- **Storage**: 50MB for app + space for downloads
- **Network**: Internet connection for downloads

### Dependencies
- **Go**: 1.21+ (for building from source)
- **Fyne**: v2.x (automatically managed)
- **System Libraries**: OpenGL, Cocoa (macOS)

## 🤝 Contributing

We welcome contributions! Here's how you can help:

1. **Fork** the repository
2. **Create** a feature branch
3. **Make** your changes
4. **Test** thoroughly
5. **Submit** a pull request

### Development Setup
```bash
# Install Go (if not already installed)
# Install dependencies
go mod tidy

# Run in development mode
go run main.go

# Build for testing
go build -o download-manager main.go
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Fyne Framework**: For the excellent cross-platform GUI framework
- **SVG Repo**: For the download cloud icon
- **Go Community**: For the amazing ecosystem and tools

## 📞 Support

If you encounter any issues or have questions:

1. **Check** the troubleshooting section above
2. **Search** existing issues in the repository
3. **Create** a new issue with detailed information
4. **Include** system information and error messages

---

**Download Manager** - Fast, reliable, and beautiful downloads for everyone! 🚀
