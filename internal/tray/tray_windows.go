//go:build windows

package tray

import (
	"context"
	"log"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Callbacks defines the callback functions for tray events
type Callbacks struct {
	OnShow func()
	OnHide func()
	OnQuit func()
}

// Manager manages the window visibility state
type Manager struct {
	mu        sync.Mutex
	ctx       context.Context
	callbacks Callbacks
	running   bool
}

const (
	NIF_MESSAGE  = 0x00000001
	NIF_ICON     = 0x00000002
	NIF_TIP      = 0x00000004
	NIF_INFO     = 0x00000010
	NIM_ADD      = 0x00000000
	NIM_MODIFY   = 0x00000001
	NIM_DELETE   = 0x00000002
	WM_USER      = 0x0400
	WM_TRAYICON  = WM_USER + 1
	WM_LBUTTONUP = 0x0202
	WM_RBUTTONUP = 0x0205
	WM_COMMAND   = 0x0111
	WM_DESTROY   = 0x0002

	IDM_SHOW = 1001
	IDM_QUIT = 1002
)

var (
	shell32               = windows.NewLazySystemDLL("shell32.dll")
	user32                = windows.NewLazySystemDLL("user32.dll")
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procShellNotifyIconW  = shell32.NewProc("Shell_NotifyIconW")
	procExtractIconExW    = shell32.NewProc("ExtractIconExW")
	procRegisterClassExW  = user32.NewProc("RegisterClassExW")
	procCreateWindowExW   = user32.NewProc("CreateWindowExW")
	procDefWindowProcW    = user32.NewProc("DefWindowProcW")
	procGetMessageW       = user32.NewProc("GetMessageW")
	procTranslateMessage  = user32.NewProc("TranslateMessage")
	procDispatchMessageW  = user32.NewProc("DispatchMessageW")
	procPostQuitMessage   = user32.NewProc("PostQuitMessage")
	procDestroyWindow     = user32.NewProc("DestroyWindow")
	procLoadIconW         = user32.NewProc("LoadIconW")
	procCreatePopupMenu   = user32.NewProc("CreatePopupMenu")
	procAppendMenuW       = user32.NewProc("AppendMenuW")
	procTrackPopupMenu    = user32.NewProc("TrackPopupMenu")
	procDestroyMenu       = user32.NewProc("DestroyMenu")
	procGetCursorPos      = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPostMessageW      = user32.NewProc("PostMessageW")
	procGetModuleFileNameW = kernel32.NewProc("GetModuleFileNameW")
)

type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             windows.Handle
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     windows.Handle
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type MSG struct {
	HWnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type POINT struct {
	X, Y int32
}

// WindowsTrayManager manages the Windows system tray icon
type WindowsTrayManager struct {
	mu        sync.Mutex
	ctx       context.Context
	callbacks Callbacks
	running   bool
	hwnd      windows.Handle
	nid       NOTIFYICONDATAW
	stopCh    chan struct{}
}

var (
	globalTrayManager *WindowsTrayManager
	globalTrayMutex   sync.Mutex
)

// NewManager creates a new tray manager (Windows implementation)
func NewManager(callbacks Callbacks) *Manager {
	return &Manager{
		callbacks: callbacks,
	}
}

// Start starts the tray manager (Windows implementation)
func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.ctx = ctx
	m.mu.Unlock()

	// Start Windows system tray
	globalTrayMutex.Lock()
	if globalTrayManager == nil {
		globalTrayManager = &WindowsTrayManager{
			callbacks: m.callbacks,
			ctx:       ctx,
			stopCh:    make(chan struct{}),
		}
		go globalTrayManager.run()
	}
	globalTrayMutex.Unlock()

	log.Println("[Tray] Windows system tray started")
}

// Stop stops the tray manager
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}
	m.running = false

	globalTrayMutex.Lock()
	if globalTrayManager != nil {
		globalTrayManager.stop()
		globalTrayManager = nil
	}
	globalTrayMutex.Unlock()

	log.Println("[Tray] Windows system tray stopped")
}

// IsRunning returns whether the tray manager is running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// ShowWindow calls the show callback
func (m *Manager) ShowWindow() {
	if m.callbacks.OnShow != nil {
		m.callbacks.OnShow()
	}
}

// HideWindow calls the hide callback
func (m *Manager) HideWindow() {
	if m.callbacks.OnHide != nil {
		m.callbacks.OnHide()
	}
}

// Quit calls the quit callback
func (m *Manager) Quit() {
	if m.callbacks.OnQuit != nil {
		m.callbacks.OnQuit()
	}
}

func (t *WindowsTrayManager) run() {
	// Register window class
	className, _ := windows.UTF16PtrFromString("PortForwardManagerTrayClass")

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   windows.NewCallback(t.wndProc),
		LpszClassName: className,
	}

	ret, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		log.Println("[Tray] Failed to register window class")
		return
	}

	// Create hidden window for tray messages
	windowName, _ := windows.UTF16PtrFromString("PortForwardManagerTray")
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0, 0, 0, 0, 0, 0, 0, 0, 0,
	)
	if hwnd == 0 {
		log.Println("[Tray] Failed to create tray window")
		return
	}
	t.hwnd = windows.Handle(hwnd)

	// Try to load icon from current executable
	icon := loadAppIcon()
	if icon == 0 {
		// Fallback to default icon
		icon, _, _ = procLoadIconW.Call(0, uintptr(32512)) // IDI_APPLICATION
	}

	// Setup notification icon data
	t.nid = NOTIFYICONDATAW{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
		HWnd:             t.hwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
		HIcon:            windows.Handle(icon),
	}

	tip := "Port Forward Manager"
	tipUtf16, _ := windows.UTF16FromString(tip)
	copy(t.nid.SzTip[:], tipUtf16)

	// Add icon to tray
	ret, _, _ = procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&t.nid)))
	if ret == 0 {
		log.Println("[Tray] Failed to add tray icon")
	}

	log.Println("[Tray] System tray icon created")

	// Message loop
	var msg MSG
	for {
		select {
		case <-t.stopCh:
			return
		default:
			ret, _, _ := procGetMessageW.Call(
				uintptr(unsafe.Pointer(&msg)),
				0, 0, 0,
			)
			if ret == 0 || int32(ret) == -1 {
				return
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func (t *WindowsTrayManager) wndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lParam {
		case WM_LBUTTONUP:
			// Left click - show window
			if t.callbacks.OnShow != nil {
				t.callbacks.OnShow()
			}
		case WM_RBUTTONUP:
			// Right click - show context menu
			t.showContextMenu()
		}
		return 0
	case WM_COMMAND:
		switch wParam {
		case IDM_SHOW:
			if t.callbacks.OnShow != nil {
				t.callbacks.OnShow()
			}
		case IDM_QUIT:
			if t.callbacks.OnQuit != nil {
				t.callbacks.OnQuit()
			}
		}
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func (t *WindowsTrayManager) showContextMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}

	showText, _ := windows.UTF16PtrFromString("Show Window")
	quitText, _ := windows.UTF16PtrFromString("Quit")

	procAppendMenuW.Call(menu, 0, IDM_SHOW, uintptr(unsafe.Pointer(showText)))
	procAppendMenuW.Call(menu, 0, IDM_QUIT, uintptr(unsafe.Pointer(quitText)))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(uintptr(t.hwnd))
	procTrackPopupMenu.Call(menu, 0, uintptr(pt.X), uintptr(pt.Y), 0, uintptr(t.hwnd), 0)
	procDestroyMenu.Call(menu)
}

func (t *WindowsTrayManager) stop() {
	// Remove tray icon
	procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&t.nid)))

	// Close stop channel
	select {
	case <-t.stopCh:
	default:
		close(t.stopCh)
	}

	// Destroy window
	if t.hwnd != 0 {
		procDestroyWindow.Call(uintptr(t.hwnd))
	}
}

// loadAppIcon loads the icon from the current executable
func loadAppIcon() uintptr {
	// Get current executable path
	buf := make([]uint16, 260)
	n, _, _ := procGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		log.Println("[Tray] Failed to get module file name")
		return 0
	}

	exePath := windows.UTF16ToString(buf[:n])

	// Extract small icon from executable
	exePathPtr, _ := windows.UTF16PtrFromString(exePath)
	var smallIcon uintptr
	ret, _, _ := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(exePathPtr)),
		0, // Icon index
		0, // Large icon (not needed)
		uintptr(unsafe.Pointer(&smallIcon)),
		1, // Number of icons to extract
	)

	if ret == 0 || smallIcon == 0 {
		log.Printf("[Tray] Failed to extract icon from %s", exePath)
		return 0
	}

	log.Printf("[Tray] Loaded app icon from %s", exePath)
	return smallIcon
}
