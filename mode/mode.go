package mode

type Mode int

const (
	Debug Mode = iota
	Prod
	Test
)

func (m Mode) String() string {
	names := [...]string{
		"Debug",
		"Prod",
		"Test",
	}

	if m < Debug || m > Test {
		return "Unknown"
	}

	return names[m]
}

var appMode = Prod

// SetMode sets the app mode according to input string.
func SetMode(mode string) {
	switch mode {
	case "debug":
		appMode = Debug
	case "prod":
		appMode = Prod
	case "test":
		appMode = Test
	}
}

// CurrentMode returns the app's current mode.
func CurrentMode() Mode {
	return appMode
}

func IsDebug() bool {
	return appMode == Debug
}

func IsProd() bool {
	return appMode == Prod
}
