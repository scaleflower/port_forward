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

// newPlatformImpl creates a new Windows implementation
func newPlatformImpl(name string) platformImpl {
	return &windowsImpl{
		name:       name,
		wakeupPort: 19847, // Fixed port for wakeup signal
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
func (w *windowsImpl) startWakeupListener(callback func()) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return nil
	}

	// Use TCP on localhost with fixed port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", w.wakeupPort))
	if err != nil {
		log.Printf("[SingleInstance] Failed to create listener on port %d: %v", w.wakeupPort, err)
		return err
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

	log.Printf("[SingleInstance] Wakeup listener started on port %d", w.wakeupPort)
	return nil
}

// sendWakeupSignal sends a wakeup signal to the existing instance
func (w *windowsImpl) sendWakeupSignal() error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", w.wakeupPort), 2*time.Second)
	if err != nil {
		log.Printf("[SingleInstance] Cannot connect to existing instance: %v", err)
		return err
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(time.Second))
	_, err = conn.Write([]byte("WAKEUP"))
	if err != nil {
		log.Printf("[SingleInstance] Failed to send wakeup signal: %v", err)
		return err
	}

	log.Println("[SingleInstance] Wakeup signal sent successfully")
	return nil
}
