package app

type App struct {
	cfg *Config
}

func New(cfg *Config) *App {
	return &App{cfg: cfg}
}
