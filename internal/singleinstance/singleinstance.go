// Package singleinstance provides cross-platform single instance detection
// and window activation for the application.
package singleinstance

// Instance represents a single instance manager
type Instance struct {
	name     string
	onWakeup func() // callback when wakeup signal received
	impl     platformImpl
}

// platformImpl is the platform-specific implementation interface
type platformImpl interface {
	tryLock() (bool, error)
	unlock() error
	startWakeupListener(callback func()) error
	sendWakeupSignal() error
}

// New creates a new single instance manager
func New(name string, onWakeup func()) *Instance {
	return &Instance{
		name:     name,
		onWakeup: onWakeup,
	}
}

// SetWakeupCallback sets the wakeup callback function
func (i *Instance) SetWakeupCallback(callback func()) {
	i.onWakeup = callback
}

// TryLock attempts to acquire the single instance lock.
// Returns true if this is the first instance, false if another instance is running.
func (i *Instance) TryLock() (bool, error) {
	i.impl = newPlatformImpl(i.name)
	return i.impl.tryLock()
}

// StartWakeupListener starts listening for wakeup signals from other instances.
// Should be called after TryLock returns true.
func (i *Instance) StartWakeupListener() error {
	if i.impl == nil {
		return nil
	}
	return i.impl.startWakeupListener(i.onWakeup)
}

// SendWakeupSignal sends a wakeup signal to the existing instance.
// Should be called when TryLock returns false.
func (i *Instance) SendWakeupSignal() error {
	if i.impl == nil {
		i.impl = newPlatformImpl(i.name)
	}
	return i.impl.sendWakeupSignal()
}

// Unlock releases the single instance lock.
// Should be called when the application is shutting down.
func (i *Instance) Unlock() error {
	if i.impl == nil {
		return nil
	}
	return i.impl.unlock()
}
