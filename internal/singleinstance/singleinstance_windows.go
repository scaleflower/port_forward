//go:build windows

package singleinstance

import (
	"fmt"
	"log"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	createMutexW = kernel32.NewProc("CreateMutexW")
	releaseMutex = kernel32.NewProc("ReleaseMutex")
	closeHandle  = kernel32.NewProc("CloseHandle")
)

const (
	ERROR_ALREADY_EXISTS = 183
)

// windowsImpl is the Windows-specific implementation
type windowsImpl struct {
	name        string
	mutexHandle syscall.Handle
	wakeupPort  int
	listener    net.Listener
	mu          sync.Mutex
	running     bool
}

// Backup ports to try if primary port is occupied
var wakeupPorts = []int{19847, 19857, 19867, 19877, 19887}

// newPlatformImpl creates a new Windows implementation
func newPlatformImpl(name string) platformImpl {
	return &windowsImpl{
		name:       name,
		wakeupPort: 19847, // Primary port for wakeup signal
	}
}

// tryLock attempts to acquire the mutex lock
func (w *windowsImpl) tryLock() (bool, error) {
	// Convert name to UTF16 for Windows API
	mutexName := "Global\\" + w.name
	namePtr, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		return false, err
	}

	// Create or open the mutex
	handle, _, callErr := createMutexW.Call(
		0, // default security attributes
		1, // initially owned
		uintptr(unsafe.Pointer(namePtr)),
	)

	if handle == 0 {
		return false, callErr
	}

	w.mutexHandle = syscall.Handle(handle)

	// Check if mutex already existed by checking GetLastError
	// The error is returned through callErr when handle != 0
	if callErr != nil && callErr.(syscall.Errno) == ERROR_ALREADY_EXISTS {
		// Another instance is running
		closeHandle.Call(uintptr(w.mutexHandle))
		w.mutexHandle = 0
		return false, nil
	}

	return true, nil
}

// unlock releases the mutex lock
func (w *windowsImpl) unlock() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.running = false

	if w.listener != nil {
		w.listener.Close()
		w.listener = nil
	}

	if w.mutexHandle != 0 {
		releaseMutex.Call(uintptr(w.mutexHandle))
		closeHandle.Call(uintptr(w.mutexHandle))
		w.mutexHandle = 0
	}

	return nil
}

// startWakeupListener starts listening for wakeup signals
// It tries multiple ports if the primary port is occupied
func (w *windowsImpl) startWakeupListener(callback func()) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return nil
	}

	// Try multiple ports in case some are occupied
	var listener net.Listener
	var err error
	var usedPort int

	for _, port := range wakeupPorts {
		listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			usedPort = port
			w.wakeupPort = port
			break
		}
		log.Printf("[SingleInstance] Port %d unavailable: %v, trying next...", port, err)
	}

	if listener == nil {
		// All ports failed, but this is non-fatal - just log and continue
		log.Printf("[SingleInstance] Warning: Could not start wakeup listener on any port. Single instance wakeup will not work.")
		return nil // Return nil to not block GUI startup
	}

	w.listener = listener
	w.running = true

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				w.mu.Lock()
				running := w.running
				w.mu.Unlock()
				if !running {
					return
				}
				continue
			}

			// Read wakeup signal
			buf := make([]byte, 16)
			conn.SetReadDeadline(time.Now().Add(time.Second))
			n, _ := conn.Read(buf)
			conn.Close()

			if n > 0 && string(buf[:n]) == "WAKEUP" {
				log.Println("[SingleInstance] Received wakeup signal")
				if callback != nil {
					callback()
				}
			}
		}
	}()

	log.Printf("[SingleInstance] Wakeup listener started on port %d", usedPort)
	return nil
}

// sendWakeupSignal sends a wakeup signal to the existing instance
// It tries multiple ports to find the listening instance
func (w *windowsImpl) sendWakeupSignal() error {
	// Try all possible ports since we don't know which one the first instance is using
	for _, port := range wakeupPorts {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		if err != nil {
			continue // Try next port
		}

		conn.SetWriteDeadline(time.Now().Add(time.Second))
		_, err = conn.Write([]byte("WAKEUP"))
		conn.Close()

		if err == nil {
			log.Printf("[SingleInstance] Wakeup signal sent successfully to port %d", port)
			return nil
		}
	}

	log.Printf("[SingleInstance] Could not send wakeup signal to any port")
	return nil // Non-fatal error
}
