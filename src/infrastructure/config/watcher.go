package config

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	// ErrWatcherNotRunning indicates watcher is not running
	ErrWatcherNotRunning = errors.New("watcher not running")
	// ErrWatcherAlreadyRunning indicates watcher is already running
	ErrWatcherAlreadyRunning = errors.New("watcher already running")
	// ErrCallbackExists indicates callback with same name already exists
	ErrCallbackExists = errors.New("callback already exists")
	// ErrCallbackNotFound indicates callback was not found
	ErrCallbackNotFound = errors.New("callback not found")
)

// ReloadCallback is a function called when configuration is reloaded
type ReloadCallback func(oldConfig, newConfig *Config) error

// ConfigWatcher watches configuration files for changes
type ConfigWatcher struct {
	filePaths        []string
	callbacks        map[string]ReloadCallback
	callbacksMutex   sync.RWMutex
	stopChan         chan bool
	running          bool
	runningMutex     sync.Mutex
	debounceInterval time.Duration
	lastReload       time.Time
	reloadMutex      sync.Mutex
	version          int
	metrics          WatcherMetrics
	metricsMutex     sync.Mutex
}

// WatcherMetrics tracks watcher operations metrics
type WatcherMetrics struct {
	SuccessfulReloads int
	FailedReloads     int
	TotalReloads      int
}

var globalWatcher *ConfigWatcher
var watcherMutex sync.Mutex

// GetWatcher returns the global config watcher instance
func GetWatcher() *ConfigWatcher {
	watcherMutex.Lock()
	defer watcherMutex.Unlock()
	
	if globalWatcher == nil {
		globalWatcher = NewWatcher()
	}
	
	return globalWatcher
}

// NewWatcher creates a new configuration watcher
func NewWatcher() *ConfigWatcher {
	return &ConfigWatcher{
		filePaths:        []string{},
		callbacks:        make(map[string]ReloadCallback),
		stopChan:         make(chan bool),
		debounceInterval: 500 * time.Millisecond,
		version:          1,
	}
}

// WatchConfigFile starts watching a configuration file for changes
func (w *ConfigWatcher) WatchConfigFile(filePath string) error {
	if filePath == "" {
		return errors.New("empty file path")
	}
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}
	
	w.runningMutex.Lock()
	defer w.runningMutex.Unlock()
	
	if w.running {
		return ErrWatcherAlreadyRunning
	}
	
	w.filePaths = append(w.filePaths, filePath)
	w.running = true
	
	go w.watchLoop()
	
	return nil
}

// WatchMultipleFiles watches multiple configuration files
func (w *ConfigWatcher) WatchMultipleFiles(filePaths []string) error {
	for _, path := range filePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
	}
	
	w.runningMutex.Lock()
	defer w.runningMutex.Unlock()
	
	if w.running {
		return ErrWatcherAlreadyRunning
	}
	
	w.filePaths = filePaths
	w.running = true
	
	go w.watchLoop()
	
	return nil
}

// StopWatching stops the file watcher
func (w *ConfigWatcher) StopWatching() error {
	w.runningMutex.Lock()
	defer w.runningMutex.Unlock()
	
	if !w.running {
		// Already stopped, not an error
		return nil
	}
	
	w.stopChan <- true
	w.running = false
	
	return nil
}

// RegisterReloadCallback registers a callback for config reloads
func (w *ConfigWatcher) RegisterReloadCallback(name string, callback ReloadCallback) error {
	if name == "" {
		return errors.New("empty callback name")
	}
	
	w.callbacksMutex.Lock()
	defer w.callbacksMutex.Unlock()
	
	if _, exists := w.callbacks[name]; exists {
		return fmt.Errorf("%w: %s", ErrCallbackExists, name)
	}
	
	w.callbacks[name] = callback
	return nil
}

// UnregisterReloadCallback unregisters a callback
func (w *ConfigWatcher) UnregisterReloadCallback(name string) error {
	w.callbacksMutex.Lock()
	defer w.callbacksMutex.Unlock()
	
	if _, exists := w.callbacks[name]; !exists {
		return fmt.Errorf("%w: %s", ErrCallbackNotFound, name)
	}
	
	delete(w.callbacks, name)
	return nil
}

// GetConfigVersion returns the current configuration version
func (w *ConfigWatcher) GetConfigVersion() int {
	w.reloadMutex.Lock()
	defer w.reloadMutex.Unlock()
	
	return w.version
}

// GetMetrics returns watcher metrics
func (w *ConfigWatcher) GetMetrics() WatcherMetrics {
	w.metricsMutex.Lock()
	defer w.metricsMutex.Unlock()
	
	return w.metrics
}

// watchLoop is the main watching loop
func (w *ConfigWatcher) watchLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	fileStates := make(map[string]time.Time)
	
	// Initialize file states
	for _, path := range w.filePaths {
		if info, err := os.Stat(path); err == nil {
			fileStates[path] = info.ModTime()
		}
	}
	
	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.checkForChanges(fileStates)
		}
	}
}

// checkForChanges checks if any watched files have changed
func (w *ConfigWatcher) checkForChanges(fileStates map[string]time.Time) {
	for _, path := range w.filePaths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		
		lastMod, exists := fileStates[path]
		if !exists || info.ModTime().After(lastMod) {
			// File changed, check debounce
			if w.shouldDebounce() {
				continue
			}
			
			fileStates[path] = info.ModTime()
			w.handleFileChange(path)
		}
	}
}

// shouldDebounce checks if we should debounce this reload
func (w *ConfigWatcher) shouldDebounce() bool {
	w.reloadMutex.Lock()
	defer w.reloadMutex.Unlock()
	
	if time.Since(w.lastReload) < w.debounceInterval {
		return true
	}
	
	return false
}

// handleFileChange handles a file change event
func (w *ConfigWatcher) handleFileChange(filePath string) {
	oldConfig := GetConfig()
	
	// Try to reload config
	newConfig, err := LoadFromFile(filePath)
	if err != nil {
		w.recordFailedReload()
		return
	}
	
	// Validate new config
	if err := ValidateCompleteConfig(newConfig); err != nil {
		w.recordFailedReload()
		return
	}
	
	// Check if config actually changed
	if !configChanged(oldConfig, newConfig) {
		return
	}
	
	// Update version
	w.reloadMutex.Lock()
	w.version++
	newConfig.version = w.version
	w.lastReload = time.Now()
	w.reloadMutex.Unlock()
	
	// Execute callbacks
	if err := w.executeCallbacks(oldConfig, newConfig); err != nil {
		// Rollback on callback failure
		w.recordFailedReload()
		return
	}
	
	// Apply new config
	SetConfig(newConfig)
	w.recordSuccessfulReload()
}

// executeCallbacks executes all registered callbacks
func (w *ConfigWatcher) executeCallbacks(oldConfig, newConfig *Config) error {
	w.callbacksMutex.RLock()
	defer w.callbacksMutex.RUnlock()
	
	for name, callback := range w.callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			return fmt.Errorf("callback %s failed: %w", name, err)
		}
	}
	
	return nil
}

// recordSuccessfulReload records a successful reload
func (w *ConfigWatcher) recordSuccessfulReload() {
	w.metricsMutex.Lock()
	defer w.metricsMutex.Unlock()
	
	w.metrics.SuccessfulReloads++
	w.metrics.TotalReloads++
}

// recordFailedReload records a failed reload
func (w *ConfigWatcher) recordFailedReload() {
	w.metricsMutex.Lock()
	defer w.metricsMutex.Unlock()
	
	w.metrics.FailedReloads++
	w.metrics.TotalReloads++
}

// configChanged checks if configuration actually changed
func configChanged(old, new *Config) bool {
	// Check reloadable fields only
	if old.Logging.Level != new.Logging.Level {
		return true
	}
	if old.Logging.Format != new.Logging.Format {
		return true
	}
	if old.Scheduler.Interval != new.Scheduler.Interval {
		return true
	}
	if old.Scheduler.BatchSize != new.Scheduler.BatchSize {
		return true
	}
	
	return false
}

// IsReloadable checks if a config field supports hot reload
func IsReloadable(field string) bool {
	reloadableFields := map[string]bool{
		"log_level":          true,
		"log_format":         true,
		"scheduler_interval": true,
		"scheduler_batch_size": true,
		"server_port":        false,
		"mongodb_uri":        false,
		"nats_url":           false,
	}
	
	return reloadableFields[field]
}

// SetDebounceInterval sets the debounce interval for file changes
func (w *ConfigWatcher) SetDebounceInterval(interval time.Duration) {
	w.reloadMutex.Lock()
	defer w.reloadMutex.Unlock()
	
	w.debounceInterval = interval
}
