//go:build !windows

package singleinstance

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// unixImpl is the Unix-specific implementation (macOS, Linux)
type unixImpl struct {
	name       string
	lockFile   *os.File
	lockPath   string
	socketPath string
	listener   net.Listener
	mu         sync.Mutex
	running    bool
}

// newPlatformImpl creates a new Unix implementation
func newPlatformImpl(name string) platformImpl {
	// Use /tmp for lock and socket files
	return &unixImpl{
		name:       name,
		lockPath:   filepath.Join("/tmp", name+".lock"),
		socketPath: filepath.Join("/tmp", name+"-wakeup.sock"),
	}
}

// tryLock attempts to acquire the file lock
func (u *unixImpl) tryLock() (bool, error) {
	// Open or create the lock file
	file, err := os.OpenFile(u.lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return false, err
	}

	// Try to acquire an exclusive lock (non-blocking)
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		if err == syscall.EWOULDBLOCK {
			// Another instance is running
			return false, nil
		}
		return false, err
	}

	// Write PID to lock file
	file.Truncate(0)
	file.Seek(0, 0)
	file.WriteString(fmt.Sprintf("%d", os.Getpid()))

	u.lockFile = file
	return true, nil
}

// unlock releases the file lock
func (u *unixImpl) unlock() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.running = false

	if u.listener != nil {
		u.listener.Close()
		u.listener = nil
	}

	// Remove socket file
	os.Remove(u.socketPath)

	if u.lockFile != nil {
		syscall.Flock(int(u.lockFile.Fd()), syscall.LOCK_UN)
		u.lockFile.Close()
		os.Remove(u.lockPath)
		u.lockFile = nil
	}

	return nil
}

// startWakeupListener starts listening for wakeup signals on a Unix socket
// This is non-fatal - if it fails, the GUI will still start
func (u *unixImpl) startWakeupListener(callback func()) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.running {
		return nil
	}

	// Remove existing socket file
	os.Remove(u.socketPath)

	listener, err := net.Listen("unix", u.socketPath)
	if err != nil {
		// Non-fatal error - just log and continue
		log.Printf("[SingleInstance] Warning: Failed to create wakeup listener: %v. Single instance wakeup will not work.", err)
		return nil // Return nil to not block GUI startup
	}

	// Set socket permissions
	os.Chmod(u.socketPath, 0666)

	u.listener = listener
	u.running = true

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				u.mu.Lock()
				running := u.running
				u.mu.Unlock()
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

	log.Printf("[SingleInstance] Wakeup listener started on %s", u.socketPath)
	return nil
}

// sendWakeupSignal sends a wakeup signal to the existing instance
// This is non-fatal - if it fails, just log and continue
func (u *unixImpl) sendWakeupSignal() error {
	conn, err := net.DialTimeout("unix", u.socketPath, 2*time.Second)
	if err != nil {
		log.Printf("[SingleInstance] Cannot connect to existing instance: %v (this is normal if wakeup listener failed)", err)
		return nil // Non-fatal
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(time.Second))
	_, err = conn.Write([]byte("WAKEUP"))
	if err != nil {
		log.Printf("[SingleInstance] Failed to send wakeup signal: %v", err)
		return nil // Non-fatal
	}

	log.Println("[SingleInstance] Wakeup signal sent successfully")
	return nil
}
