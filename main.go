package main

import (
	"claude-squad/app"
	"claude-squad/logger"
	"context"
)

func main() {
	ctx := context.Background()
	logger.Initialize()
	defer logger.Close()

	app.Run(ctx)
	//tmuxMain()
}
