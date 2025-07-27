package tools

import (
	"context"
	"path/filepath"
	"plugin"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// PluginLoader handles loading custom tools
type PluginLoader struct {
	searchPaths   []string
	loadedPlugins map[string]*LoadedPlugin
}

// LoadedPlugin represents a loaded plugin instance
type LoadedPlugin struct {
	Path     string
	Plugin   ToolPlugin
	Metadata PluginMetadata
	Enabled  bool
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(searchPaths []string) *PluginLoader {
	return &PluginLoader{
		searchPaths:   searchPaths,
		loadedPlugins: make(map[string]*LoadedPlugin),
	}
}

// LoadPlugins discovers and loads all plugins from search paths
func (pl *PluginLoader) LoadPlugins() error {
	for _, searchPath := range pl.searchPaths {
		// Find .so files (compiled Go plugins)
		matches, err := filepath.Glob(filepath.Join(searchPath, "*.so"))
		if err != nil {
			logger.LogErr(err, "failed to search for plugins", "path", searchPath)
			continue
		}

		for _, pluginPath := range matches {
			if err := pl.loadPlugin(pluginPath); err != nil {
				logger.LogErr(err, "failed to load plugin", "path", pluginPath)
				// Continue loading other plugins
			}
		}
	}
	return nil
}

// loadPlugin loads a single plugin
func (pl *PluginLoader) loadPlugin(path string) error {
	// Load the Go plugin
	p, err := plugin.Open(path)
	if err != nil {
		return serr.Wrap(err, "failed to open plugin")
	}

	// Look for the required symbol
	sym, err := p.Lookup("Tool")
	if err != nil {
		return serr.Wrap(err, "plugin missing 'Tool' symbol")
	}

	// Assert to ToolPlugin interface
	toolPlugin, ok := sym.(ToolPlugin)
	if !ok {
		return serr.New("plugin 'Tool' does not implement ToolPlugin interface")
	}

	// Get plugin metadata
	metaSym, err := p.Lookup("Metadata")
	if err != nil {
		return serr.Wrap(err, "plugin missing 'Metadata' symbol")
	}

	metadata, ok := metaSym.(*PluginMetadata)
	if !ok {
		return serr.New("plugin 'Metadata' is not of type *PluginMetadata")
	}

	// Initialize the plugin
	if err := toolPlugin.Initialize(nil); err != nil {
		return serr.Wrap(err, "plugin initialization failed")
	}

	// Store the loaded plugin
	pl.loadedPlugins[metadata.Name] = &LoadedPlugin{
		Path:     path,
		Plugin:   toolPlugin,
		Metadata: *metadata,
		Enabled:  true,
	}

	logger.Info("Loaded custom tool plugin",
		"name", metadata.Name,
		"version", metadata.Version,
		"author", metadata.Author)

	return nil
}

// GetPlugins returns all loaded plugins
func (pl *PluginLoader) GetPlugins() map[string]*LoadedPlugin {
	return pl.loadedPlugins
}

// RegisterWithRegistry adds all loaded plugins to a registry
func (pl *PluginLoader) RegisterWithRegistry(registry *Registry, projectRoot string) error {
	for name, loadedPlugin := range pl.loadedPlugins {
		if !loadedPlugin.Enabled {
			continue
		}

		// Create an executor adapter
		executor := &PluginExecutorAdapter{
			plugin: loadedPlugin.Plugin,
		}

		// Wrap with sandbox for security
		sandboxedExecutor := WrapWithSandbox(executor, loadedPlugin.Plugin, projectRoot)

		// Register with the registry
		registry.Register(loadedPlugin.Plugin.GetDefinition(), sandboxedExecutor)

		logger.Debug("Registered custom tool with registry", "tool", name)
	}
	return nil
}

// PluginExecutorAdapter adapts a ToolPlugin to the Executor interface
type PluginExecutorAdapter struct {
	plugin ToolPlugin
}

// Execute implements the Executor interface
func (a *PluginExecutorAdapter) Execute(input map[string]interface{}) (string, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Execute the plugin
	return a.plugin.Execute(ctx, input)
}
