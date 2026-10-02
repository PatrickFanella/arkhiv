package recovery

import "git.subcult.tv/subculture-collective/arkhiv/internal/config"

func writeJSON(path string, v any) error { return config.SaveJSON(path, v) }
