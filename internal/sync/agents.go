package sync

import "proman/internal/config"

func syncAgents(source config.Tool, target config.Tool) (int, error) {
	return mirrorDirectory(source.AgentDirPath(), target.AgentDirPath())
}
