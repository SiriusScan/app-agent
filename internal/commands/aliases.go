package commands

// init registers all built-in command aliases.
// This provides user-friendly short forms for longer internal commands.
func init() {
	// Note: Aliases are registered after all commands are registered.
	// The init() functions run in the order files are compiled, so we use
	// a separate registration function that's called after command registration.
}

// RegisterBuiltinAliases registers all standard command aliases.
// This should be called after all commands have been registered.
func RegisterBuiltinAliases() {
	// Template scanning aliases
	safeRegisterAlias("scan", "internal:template-scan")
	safeRegisterAlias("template-scan", "internal:template-scan")
	safeRegisterAlias("tscan", "internal:template-scan")

	// Software inventory/scan aliases
	safeRegisterAlias("inventory", "internal:scan")
	safeRegisterAlias("software", "internal:scan")

	// Status aliases
	safeRegisterAlias("status", "internal:status")
	safeRegisterAlias("info", "internal:status")

	// Repository aliases
	safeRegisterAlias("repo", "internal:repo")
	safeRegisterAlias("repository", "internal:repo")

	// Sync aliases (if sync command exists)
	safeRegisterAlias("sync", "internal:sync")

	// Template discovery/list aliases
	safeRegisterAlias("list-templates", "internal:list-templates")
	safeRegisterAlias("templates", "internal:list-templates")
	safeRegisterAlias("discover-templates", "internal:discover-templates")
	safeRegisterAlias("discover", "internal:discover-templates")
}

// safeRegisterAlias registers an alias only if the target command exists.
// This prevents panics if a command isn't compiled in or doesn't exist yet.
func safeRegisterAlias(alias, canonicalPrefix string) {
	registry := DefaultRegistry()

	// Check if the canonical prefix exists
	if _, exists := registry.Get(canonicalPrefix); exists {
		// Only register if not already registered
		if !registry.HasAlias(alias) {
			// Also check it doesn't conflict with existing prefixes
			if !registry.HasCommand(alias) {
				registry.RegisterAlias(alias, canonicalPrefix)
			}
		}
	}
	// Silently skip if target doesn't exist or alias conflicts
}
