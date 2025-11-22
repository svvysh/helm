package config

// DefaultEnvironment indicates the default runtime environment used in tests.
const DefaultEnvironment = "development"

// CurrentEnvironment returns the environment Helm should assume when no overrides exist.
func CurrentEnvironment() string {
	return DefaultEnvironment
}
